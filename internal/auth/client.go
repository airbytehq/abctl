package auth

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Client provides an HTTP client with automatic token management.
// Thread-safe for concurrent use by multiple goroutines/services.
// The mutex (mu) protects credentials during refresh operations when
// multiple API calls might trigger refresh simultaneously.
type Client struct {
	httpClient  *http.Client
	provider    *Provider
	clientID    string
	credentials *Credentials
	mu          sync.RWMutex // Protects credentials for concurrent access
}

// NewClient creates a new authenticated HTTP client.
// Designed for reuse across multiple services making concurrent API calls.
func NewClient(provider *Provider, clientID string, credentials *Credentials) *Client {
	return &Client{
		httpClient:  http.DefaultClient,
		provider:    provider,
		clientID:    clientID,
		credentials: credentials,
	}
}

// Do performs an authenticated HTTP request with automatic token refresh.
// Thread-safe and handles concurrent requests. If token is expired,
// it will refresh once and retry the request.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	// Ensure we have a valid token
	if err := c.ensureValidToken(req.Context()); err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	// Add authorization header
	c.mu.RLock()
	tokenType := c.credentials.TokenType
	if tokenType == "" {
		tokenType = "Bearer" // Default token type
	}
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", tokenType, c.credentials.AccessToken))
	c.mu.RUnlock()

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// If we get a 401, try refreshing the token once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		// Force refresh by marking token as expired
		c.mu.Lock()
		c.credentials.ExpiresAt = time.Now().Add(-time.Hour)
		c.mu.Unlock()

		// Try refreshing
		if err := c.ensureValidToken(req.Context()); err != nil {
			return nil, fmt.Errorf("failed to refresh token after 401: %w", err)
		}

		// Retry the request with new token
		c.mu.RLock()
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", tokenType, c.credentials.AccessToken))
		c.mu.RUnlock()

		return c.httpClient.Do(req)
	}

	return resp, nil
}

// GetCredentials returns a copy of the current credentials
func (c *Client) GetCredentials() *Credentials {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to prevent external mutation
	creds := *c.credentials
	return &creds
}

// ensureValidToken checks if the current token is valid and refreshes if needed
func (c *Client) ensureValidToken(ctx context.Context) error {
	c.mu.RLock()
	expired := c.credentials.IsExpired()
	hasRefresh := c.credentials.RefreshToken != ""
	c.mu.RUnlock()

	if !expired {
		return nil
	}

	if !hasRefresh {
		return fmt.Errorf("access token expired and no refresh token available")
	}

	// Acquire write lock and double-check
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.credentials.IsExpired() {
		// Another goroutine already refreshed
		return nil
	}

	// Refresh the token
	tokens, err := RefreshAccessToken(ctx, c.provider, c.clientID, c.credentials.RefreshToken)
	if err != nil {
		return fmt.Errorf("token refresh failed: %w", err)
	}

	// Update credentials
	c.credentials.AccessToken = tokens.AccessToken
	if tokens.RefreshToken != "" {
		c.credentials.RefreshToken = tokens.RefreshToken
	}
	if tokens.TokenType != "" {
		c.credentials.TokenType = tokens.TokenType
	}

	// Update expiration
	if tokens.ExpiresIn > 0 {
		c.credentials.ExpiresAt = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	}

	return nil
}
