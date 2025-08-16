package oidc

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// AuthenticatedClient wraps an HTTP client with automatic token management
type AuthenticatedClient struct {
	httpClient   *http.Client
	credentials  *Credentials
	provider     *ProviderConfig
	clientID     string
	mu           sync.RWMutex
}

// NewAuthenticatedClient creates a new authenticated HTTP client
func NewAuthenticatedClient(creds *Credentials, provider *ProviderConfig, clientID string) *AuthenticatedClient {
	return &AuthenticatedClient{
		httpClient:  http.DefaultClient,
		credentials: creds,
		provider:    provider,
		clientID:    clientID,
	}
}

// Do performs an authenticated HTTP request
func (c *AuthenticatedClient) Do(req *http.Request) (*http.Response, error) {
	// Check if token needs refresh
	if err := c.ensureValidToken(req.Context()); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}
	
	// Add authorization header
	c.mu.RLock()
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.credentials.TokenType, c.credentials.AccessToken))
	c.mu.RUnlock()
	
	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	
	// If unauthorized, try refreshing token once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		
		// Force token refresh
		c.mu.Lock()
		c.credentials.ExpiresAt = c.credentials.ExpiresAt.AddDate(-1, 0, 0) // Mark as expired
		c.mu.Unlock()
		
		if err := c.ensureValidToken(req.Context()); err != nil {
			return nil, fmt.Errorf("failed to refresh after 401: %w", err)
		}
		
		// Retry request with new token
		c.mu.RLock()
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.credentials.TokenType, c.credentials.AccessToken))
		c.mu.RUnlock()
		
		return c.httpClient.Do(req)
	}
	
	return resp, nil
}

// ensureValidToken checks if token is valid and refreshes if needed
func (c *AuthenticatedClient) ensureValidToken(ctx context.Context) error {
	c.mu.RLock()
	expired := c.credentials.IsExpired()
	hasRefreshToken := c.credentials.RefreshToken != ""
	c.mu.RUnlock()
	
	if !expired {
		return nil
	}
	
	if !hasRefreshToken {
		return fmt.Errorf("access token expired and no refresh token available")
	}
	
	// Refresh token
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Double-check after acquiring write lock
	if !c.credentials.IsExpired() {
		return nil
	}
	
	tokens, err := RefreshToken(ctx, c.provider, c.clientID, c.credentials.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	
	// Update credentials
	c.credentials.AccessToken = tokens.AccessToken
	if tokens.RefreshToken != "" {
		c.credentials.RefreshToken = tokens.RefreshToken
	}
	if tokens.ExpiresIn > 0 {
		c.credentials.ExpiresAt = c.credentials.ExpiresAt.Add(time.Duration(tokens.ExpiresIn) * time.Second)
	}
	
	// Save updated credentials
	if err := SaveCredentials(c.credentials); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to save refreshed credentials: %v\n", err)
	}
	
	return nil
}

// GetCredentials returns the current credentials
func (c *AuthenticatedClient) GetCredentials() *Credentials {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.credentials
}