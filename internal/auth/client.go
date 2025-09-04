package auth

import (
	"context"
	"fmt"
	stdhttp "net/http"
	"sync"
	"time"

	"github.com/airbytehq/abctl/internal/http"
)

// ErrNoUpdateHook is returned when no update hook is provided
var ErrNoUpdateHook = fmt.Errorf("no credentials update hook")

// CredentialsUpdateHook is called when credentials are refreshed
type CredentialsUpdateHook func(credentials *Credentials) error

// Client provides an HTTP client with automatic token management.
// Thread-safe for concurrent use by multiple goroutines/services.
// The mutex (mu) protects credentials during refresh operations when
// multiple API calls might trigger refresh simultaneously.
type Client struct {
	httpClient  http.HTTPDoer
	provider    *Provider
	clientID    string
	credentials *Credentials
	updateHook  CredentialsUpdateHook
	mu          sync.RWMutex // Protects credentials for concurrent access
}

// NewClient creates a new authenticated HTTP client.
// Designed for reuse across multiple services making concurrent API calls.
// If updateHook is nil, uses default hook that returns ErrNoUpdateHook.
func NewClient(
	provider *Provider,
	clientID string,
	credentials *Credentials,
	httpDoer http.HTTPDoer,
	updateHook CredentialsUpdateHook,
) *Client {
	if updateHook == nil {
		updateHook = func(*Credentials) error { return ErrNoUpdateHook }
	}
	return &Client{
		httpClient:  httpDoer,
		provider:    provider,
		clientID:    clientID,
		credentials: credentials,
		updateHook:  updateHook,
	}
}

// Do performs an authenticated HTTP request with automatic token refresh.
// Thread-safe and handles concurrent requests. If token is expired,
// it will refresh once and retry the request.
func (c *Client) Do(req *stdhttp.Request) (*stdhttp.Response, error) {
	// Ensure we have a valid token
	if err := c.ensureValidToken(req.Context()); err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	// Add authorization header
	c.mu.RLock()
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.credentials.TokenType, c.credentials.AccessToken))
	c.mu.RUnlock()

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// If we get a 401, try refreshing the token once
	if resp.StatusCode == stdhttp.StatusUnauthorized {
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
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.credentials.TokenType, c.credentials.AccessToken))
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
		return fmt.Errorf("not authenticated - please run 'airbox auth login' first")
	}

	// Acquire write lock and double-check
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.credentials.IsExpired() {
		// Another goroutine already refreshed
		return nil
	}

	// Refresh the token
	tokens, err := RefreshAccessToken(ctx, c.provider, c.clientID, c.credentials.RefreshToken, c.httpClient)
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
	} else {
		c.credentials.TokenType = "Bearer" // Default to Bearer if not specified
	}

	// Update expiration
	if tokens.ExpiresIn > 0 {
		c.credentials.ExpiresAt = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	}

	// Call update hook with refreshed credentials
	if err := c.updateHook(c.credentials); err != nil {
		return fmt.Errorf("credentials update hook failed: %w", err)
	}

	return nil
}
