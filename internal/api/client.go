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

// doRequest performs an authenticated API request to the public API
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	return c.doRequestWithQuery(ctx, method, endpoint, nil, body)
}

// doRequestWithQuery performs an authenticated API request with query parameters
func (c *Client) doRequestWithQuery(ctx context.Context, method, endpoint string, queryParams url.Values, body interface{}) (*http.Response, error) {
	// Build full URL
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = path.Join(u.Path, "api", "public", "v1", endpoint)
	
	// Add query parameters if provided
	if queryParams != nil {
		u.RawQuery = queryParams.Encode()
	}
	
	// Serialize body if provided
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		// Debug: Print the JSON being sent
		fmt.Printf("DEBUG - Request body JSON: %s\n", string(jsonBody))
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

// doRequestWithPath performs an authenticated API request with a custom path
func (c *Client) doRequestWithPath(ctx context.Context, method, fullPath string, body interface{}) (*http.Response, error) {
	// Build full URL
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = path.Join(u.Path, fullPath)
	
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
	
	// Read the full response body for better error reporting
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}
	
	if v != nil {
		if err := json.Unmarshal(body, v); err != nil {
			return fmt.Errorf("failed to decode response (status %d): %w\nResponse body: %s", 
				resp.StatusCode, err, string(body))
		}
	}
	
	return nil
}