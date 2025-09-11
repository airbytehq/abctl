package airbox

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/airbytehq/abctl/internal/api"
	"github.com/airbytehq/abctl/internal/auth"
	"github.com/airbytehq/abctl/internal/http"
)

// APIServiceFactory creates authenticated API services with all boilerplate handled
type APIServiceFactory func(ctx context.Context, httpClient http.HTTPDoer, cfg ConfigStore) (api.Service, error)

// NewAPIService creates an authenticated API service with all boilerplate handled
func NewAPIService(ctx context.Context, httpClient http.HTTPDoer, cfg ConfigStore) (api.Service, error) {
	// Load airbox config
	abCfg, err := cfg.Load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, NewConfigInitError("no airbox configuration found")
		}
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Check credentials exist
	if abCfg.Credentials == nil {
		return nil, NewLoginError("no credentials")
	}

	// Get the current context set in the Airbox config
	currentContext, err := abCfg.GetCurrentContext()
	if err != nil {
		return nil, fmt.Errorf("no current context configured: %w", err)
	}

	// Create HTTP client with API base URL from config
	apiHTTPClient, err := http.NewClient(currentContext.AirbyteAPIURL, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Check if auth is configured
	if currentContext.Auth.provider == nil {
		return nil, NewConfigInitError("no auth configured")
	}

	// Create auth provider based on auth type from config (NO HTTP CALLS)
	var authProvider auth.Provider
	switch currentContext.Auth.provider.Type() {
	case auth.OAuth2ProviderName:
		oauth, err := currentContext.Auth.GetOAuth2Provider()
		if err != nil {
			panic(err)
		}

		// Create OAuth2 provider WITHOUT authenticating
		store := NewCredentialStoreAdapter(cfg)
		authProvider, err = auth.NewOAuth2Provider(
			ctx,
			oauth.ClientID,
			oauth.ClientSecret,
			apiHTTPClient,
			store,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OAuth2 provider: %w", err)
		}

	case auth.OIDCProviderName:
		// For OIDC, create provider with stored credentials
		store := NewCredentialStoreAdapter(cfg)
		authProvider = auth.NewOIDCProvider(apiHTTPClient, store)

	default:
		return nil, fmt.Errorf("unsupported auth type: %s", currentContext.Auth.provider.Type())
	}

	// Create and return API service
	return api.NewClient(authProvider), nil
}
