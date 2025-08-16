package oidc

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
)

// DefaultClientID is the default OIDC client ID for public clients
const DefaultClientID = "abctl"

// Login performs the full login flow and saves credentials
func Login(ctx context.Context, baseURL, oidcServer string) error {
	// Discover OIDC provider
	pterm.Info.Println("Discovering OIDC provider configuration...")
	provider, err := DiscoverProvider(ctx, oidcServer)
	if err != nil {
		return fmt.Errorf("failed to discover provider: %w", err)
	}
	
	// Create auth flow
	flow, err := NewAuthFlow(provider, DefaultClientID)
	if err != nil {
		return fmt.Errorf("failed to create auth flow: %w", err)
	}
	
	// Perform authentication
	pterm.Info.Println("Starting authentication flow...")
	tokens, err := flow.Authenticate(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	
	// Create and save credentials
	creds := NewCredentials(tokens, baseURL, oidcServer)
	if err := SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}
	
	pterm.Success.Println("Authentication successful! Credentials saved.")
	return nil
}

// GetAuthenticatedClient loads credentials and returns an authenticated HTTP client
func GetAuthenticatedClient(ctx context.Context) (*AuthenticatedClient, error) {
	// Load credentials
	creds, err := LoadCredentials()
	if err != nil {
		return nil, err
	}
	
	// Discover provider for token refresh
	provider, err := DiscoverProvider(ctx, creds.OIDCServer)
	if err != nil {
		return nil, fmt.Errorf("failed to discover provider: %w", err)
	}
	
	// Create authenticated client
	client := NewAuthenticatedClient(creds, provider, DefaultClientID)
	return client, nil
}

// Logout removes stored credentials
func Logout() error {
	if err := DeleteCredentials(); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}
	
	pterm.Success.Println("Successfully logged out.")
	return nil
}