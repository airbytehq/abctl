package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/api"
	"github.com/airbytehq/abctl/internal/auth"
	"github.com/airbytehq/abctl/internal/http"
	"github.com/airbytehq/abctl/internal/ui"
)

// LoginCmd handles application credentials login
type LoginCmd struct {
	Namespace string `short:"n" help:"Target namespace (default: current kubeconfig context)."`
}

// Run executes the login command
func (c *LoginCmd) Run(ctx context.Context, cfgStore airbox.ConfigStore, httpClient http.HTTPDoer, apiFactory airbox.APIServiceFactory, ui ui.Provider) error {
	ui.Title("Authenticating with Airbyte")

	// Load airbox config to get current context
	cfg, err := cfgStore.Load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return airbox.NewConfigInitError("no airbox configuration found")
		}
		return fmt.Errorf("failed to load airbox config: %w", err)
	}

	// Get current context
	currentContext, err := cfg.GetCurrentContext()
	if err != nil {
		return airbox.NewConfigInitError(err.Error())
	}

	ui.ShowInfo("Connecting to: " + currentContext.AirbyteAPIURL)
	ui.NewLine()

	// Validate auth configuration from context
	if err := currentContext.Auth.Validate(); err != nil {
		return airbox.NewConfigInitError("invalid auth configuration")
	}

	// Create auth client based on configured auth method
	var authClient auth.Provider

	// Create credentials store adapter
	credStore := airbox.NewCredentialStoreAdapter(cfgStore)

	switch provider := currentContext.Auth.GetProvider().(type) {
	case *airbox.OAuth2:
		// OAuth2 (Application) - needs API client with base URL
		apiHTTPClient, err := http.NewClient(currentContext.AirbyteAPIURL, httpClient)
		if err != nil {
			return err
		}
		authClient, err = auth.NewOAuth2Provider(ctx, provider.ClientID, provider.ClientSecret, apiHTTPClient, credStore)
		if err != nil {
			return fmt.Errorf("failed to create OAuth2 client: %w", err)
		}

	case *airbox.OIDC:
		// OIDC (Email/Password or SSO) - needs plain HTTP client since auth URL is complete
		ui.ShowInfo(fmt.Sprintf("Discovering OIDC provider at: %s", provider.AuthURL))
		discoveredProvider, err := auth.DiscoverProvider(ctx, provider.AuthURL, httpClient)
		if err != nil {
			return fmt.Errorf("failed to discover OIDC provider at %s: %w", provider.AuthURL, err)
		}
		// DiscoverProvider only fetches metadata but doesn't set a store for saving tokens
		// We need to set it so the provider can persist credentials after authentication
		discoveredProvider.SetCredentialsStore(credStore)

		// Get the client ID from our saved config
		// provider is already *airbox.OIDC from the type switch

		// Run the OIDC authentication flow to get credentials
		creds, err := performOIDCFlow(ctx, discoveredProvider, provider.ClientID, httpClient, ui)
		if err != nil {
			return fmt.Errorf("OIDC authentication failed: %w", err)
		}

		// Save credentials and create provider
		if err := credStore.Save(creds); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}
		authClient = discoveredProvider

	default:
		return fmt.Errorf("unknown auth provider type: %T", provider)
	}

	// For OAuth2, provider already fetched tokens during creation
	// For OIDC, we already have credentials from the flow above
	var creds *auth.Credentials
	switch currentContext.Auth.GetProvider().(type) {
	case *airbox.OAuth2:
		creds, err = authClient.Load()
		if err != nil {
			return fmt.Errorf("failed to load credentials: %w", err)
		}
	case *airbox.OIDC:
		// Credentials were already saved during OIDC flow, just load them
		creds, err = authClient.Load()
		if err != nil {
			return fmt.Errorf("failed to load credentials: %w", err)
		}
	}
	cfg.Credentials = creds

	// Store credentials in local config
	if err := cfgStore.Save(cfg); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	ui.ShowSuccess("Successfully authenticated!")
	ui.NewLine()

	// Create API client for workspace operations
	apiClient, err := apiFactory(ctx, httpClient, cfgStore)
	if err != nil {
		return err
	}

	// Step 3: Organization Selection
	var org *api.Organization
	orgs, err := apiClient.ListOrganizations(ctx)
	if err != nil {
		return err
	}

	// Sort organizations by name
	sort.Slice(orgs, func(i, j int) bool {
		return orgs[i].Name < orgs[j].Name
	})

	if len(orgs) > 1 {
		orgNames := make([]string, len(orgs))
		for i, org := range orgs {
			orgNames[i] = org.Name
		}

		// Use filterable select for better UX when there are many organizations
		var idx int
		if len(orgs) > 10 {
			idx, _, err = ui.FilterableSelect("Select an organization", orgNames)
		} else {
			idx, _, err = ui.Select("Select an organization:", orgNames)
		}
		if err != nil {
			return err
		}

		org = orgs[idx]
	} else {
		org = orgs[0]
	}

	currentContext.OrganizationID = org.ID

	// Save the updated context (with selected org)
	cfg.AddContext(cfg.CurrentContext, *currentContext)
	if err := cfgStore.Save(cfg); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	return nil
}

// performOIDCFlow runs the browser-based OIDC authentication flow
func performOIDCFlow(ctx context.Context, provider *auth.OIDCProvider, clientID string, httpClient http.HTTPDoer, ui ui.Provider) (*auth.Credentials, error) {
	// Create flow with the client ID from config and discovered provider metadata
	flow := auth.NewFlow(clientID, 0, httpClient, auth.WithProvider(provider))

	// Start callback server
	if err := flow.StartCallbackServer(); err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}

	ui.ShowInfo("Opening browser for authentication...")

	// Send auth request (opens browser)
	if err := flow.SendAuthRequest(); err != nil {
		return nil, fmt.Errorf("failed to send auth request: %w", err)
	}

	// Wait for callback with credentials
	creds, err := flow.WaitForCallback(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	return creds, nil
}
