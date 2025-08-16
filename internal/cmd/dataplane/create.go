package dataplane

import (
	"context"
	"fmt"
	"strings"

	"github.com/airbytehq/abctl/internal/auth/oidc"
	"github.com/pterm/pterm"
)

type CreateCmd struct {
}

func (c *CreateCmd) Run() error {
	// Prompt for base URL with default value
	defaultURL := "https://cloud.airbyte.com"

	urlPrompt := fmt.Sprintf("Enter the base URL of the Airbyte instance [%s]", defaultURL)
	baseURL, _ := pterm.DefaultInteractiveTextInput.Show(urlPrompt)

	// Use default if user didn't enter anything
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultURL
	}

	pterm.Info.Printf("Using Airbyte instance at: %s\n", baseURL)

	// Prompt for OIDC server root with default value
	defaultOIDCServer := "https://cloud.airbyte.com/auth"
	
	oidcPrompt := fmt.Sprintf("Enter the OIDC server root [%s]", defaultOIDCServer)
	oidcServer, _ := pterm.DefaultInteractiveTextInput.Show(oidcPrompt)
	
	// Use default if user didn't enter anything
	oidcServer = strings.TrimSpace(oidcServer)
	if oidcServer == "" {
		oidcServer = defaultOIDCServer
	}
	
	pterm.Info.Printf("Using OIDC server at: %s\n", oidcServer)

	// Perform OIDC authentication
	ctx := context.Background()
	if err := oidc.Login(ctx, baseURL, oidcServer); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	
	// Get authenticated client for future API calls
	client, err := oidc.GetAuthenticatedClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authenticated client: %w", err)
	}
	
	// Example: You can now use the client to make authenticated API calls
	pterm.Info.Printf("Successfully authenticated. Ready to make API calls to %s\n", client.GetCredentials().BaseURL)
	
	return nil
}
