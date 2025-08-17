package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/airbytehq/abctl/internal/auth/oidc"
)

// Client is the Airbyte API client
type Client struct {
	baseURL    string
	authClient *oidc.AuthenticatedClient
}

// NewClient creates a new Airbyte API client
func NewClient(baseURL string, authClient *oidc.AuthenticatedClient) *Client {
	return &Client{
		baseURL:    baseURL,
		authClient: authClient,
	}
}

// doRequest performs an authenticated API request
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	// Build full URL
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = path.Join(u.Path, "api", "public", "v1", endpoint)
	
	// Serialize body if provided
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set content type for requests with body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	// The AuthenticatedClient will automatically add the Authorization header
	// with the bearer token and handle token refresh if needed
	return c.authClient.Do(req)
}

// parseResponse parses the API response into the provided interface
func parseResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}
	
	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	
	return nil
}