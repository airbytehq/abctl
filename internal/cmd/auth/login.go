package auth

import (
	"context"
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"os"
	"strings"
	"time"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/api"
	"github.com/airbytehq/abctl/internal/http"
	"github.com/airbytehq/abctl/internal/ui"
)

// LoginCmd handles application credentials login
type LoginCmd struct {
	Namespace string `short:"n" help:"Target namespace (default: current kubeconfig context)."`
}

// Run executes the login command
func (c *LoginCmd) Run(ctx context.Context, httpClient http.HTTPDoer, cfg airbox.ConfigProvider, ui ui.Provider) error {
	ui.Title("Authenticating with Airbyte")

	// Get client credentials from environment variables
	clientID := os.Getenv("AIRBYTE_CLIENT_ID")
	clientSecret := os.Getenv("AIRBYTE_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("client credentials required: set AIRBYTE_CLIENT_ID and AIRBYTE_CLIENT_SECRET environment variables")
	}

	// Load airbox config to get current context
	abCfg, err := cfg.Load()
	if err != nil {
		return fmt.Errorf("failed to load airbox config: %w", err)
	}

	// Get current context
	currentContext, err := abCfg.GetCurrentContext()
	if err != nil {
		return fmt.Errorf("no airbox configuration found - please run 'airbox config init' first: %w", err)
	}

	ui.ShowInfo("Connecting to: " + currentContext.AirbyteAPIHost)
	ui.NewLine()

	// Make token request to /v1/applications/token
	credentials, err := c.authenticateWithApplicationCredentials(ctx, httpClient, currentContext.AirbyteAPIHost, clientID, clientSecret)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Store credentials in airbox config
	abCfg.Credentials = credentials

	// Store credentials in local config
	if err := cfg.Save(abCfg); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	ui.ShowInfo("Successfully authenticated!")
	ui.NewLine()

	// Create API client for workspace operations
	apiClient, err := api.NewFactory(ctx, cfg, httpClient)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	var org *api.Organization
	orgs, err := apiClient.ListOrganizations(ctx)
	if err != nil {
		return err
	}

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

	abCfg.AddContext(abCfg.CurrentContext, *currentContext)
	if err := cfg.Save(abCfg); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	err = cfg.Save(abCfg)
	if err != nil {
		return err
	}

	return nil
}

func (c *LoginCmd) authenticateWithApplicationCredentials(ctx context.Context, httpClient http.HTTPDoer, apiHost, clientID, clientSecret string) (*airbox.Credentials, error) {
	// Make token request to /v1/applications/token
	tokenURL := apiHost + "/v1/applications/token"

	requestBody := fmt.Sprintf(`{"client_id": "%s", "client_secret": "%s"}`, clientID, clientSecret)

	req, err := stdhttp.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("token request failed with status %d", resp.StatusCode)
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Convert to airbox credentials
	credentials := &airbox.Credentials{
		AccessToken: tokenResponse.AccessToken,
		TokenType:   "Bearer", // Application credentials always use Bearer tokens
	}

	// Set expiry if provided
	if tokenResponse.ExpiresIn > 0 {
		credentials.ExpiresAt = time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)
	}

	return credentials, nil
}
