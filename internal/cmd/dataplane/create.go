package dataplane

import (
	"fmt"
	"strings"

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

	return nil
}
