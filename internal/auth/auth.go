package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	stdhttp "net/http"
	"net/url"
	"strings"
	"time"

	"github.com/airbytehq/abctl/internal/http"
)

const (
	// defaultTokenExpiry is the fallback expiration time for tokens without explicit expiry
	defaultTokenExpiry = time.Hour
	// clockSkewBuffer accounts for clock differences between client and server
	clockSkewBuffer = time.Minute
	// defaultAuthTimeout is the maximum time to wait for authentication flow completion
	defaultAuthTimeout = 5 * time.Minute
)

// CredentialsStore interface for storing/loading credentials
type CredentialsStore interface {
	Load() (*Credentials, error)
	Save(*Credentials) error
}

// Provider combines HTTP client functionality with credential storage for authenticated requests
type Provider interface {
	http.HTTPDoer
	CredentialsStore
}

// TokenResponse represents an OAuth2 token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// oAuth2ErrorResponse represents a standard OAuth2 error response as defined in RFC 6749
type oAuth2ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// Credentials stores authentication tokens and metadata
type Credentials struct {
	AccessToken  string    `json:"access_token" yaml:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty" yaml:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type" yaml:"token_type"`
	ExpiresAt    time.Time `json:"expires_at" yaml:"expires_at"`
}

// IsExpired checks if the access token has expired
func (c *Credentials) IsExpired() bool {
	// Consider expired 1 minute early to account for clock skew
	return time.Now().After(c.ExpiresAt.Add(-clockSkewBuffer))
}

// credentialsFromJSON deserializes credentials from JSON
func credentialsFromJSON(data []byte) (*Credentials, error) {
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

// DiscoverProvider fetches OIDC provider configuration from well-known endpoint
func DiscoverProvider(ctx context.Context, issuerURL string, client http.HTTPDoer) (*OIDCProvider, error) {
	return discoverProviderWithClient(ctx, issuerURL, client)
}

// discoverProviderWithClient fetches OIDC provider configuration using the provided HTTP client
func discoverProviderWithClient(ctx context.Context, issuerURL string, client http.HTTPDoer) (*OIDCProvider, error) {
	// Construct well-known URL
	wellKnownURL := fmt.Sprintf("%s/.well-known/openid-configuration", issuerURL)

	req, err := stdhttp.NewRequestWithContext(ctx, "GET", wellKnownURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch provider configuration: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() // Connection cleanup, error doesn't affect functionality

	if resp.StatusCode != stdhttp.StatusOK {
		return nil, fmt.Errorf("discovery failed with status %d", resp.StatusCode)
	}

	// Read response body for better error reporting
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read provider configuration response: %w", err)
	}

	var provider OIDCProvider
	if err := json.Unmarshal(body, &provider); err != nil {
		return nil, fmt.Errorf("failed to decode provider configuration: %w", err)
	}

	return &provider, nil
}

func doTokenRequest(ctx context.Context, endpoint string, data url.Values, client http.HTTPDoer) (*TokenResponse, error) {
	body := strings.NewReader(data.Encode())
	req, err := stdhttp.NewRequestWithContext(ctx, "POST", endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() // Connection cleanup, error doesn't affect functionality

	if resp.StatusCode != stdhttp.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		// Try standard OAuth2 error format
		var errResp oAuth2ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("failed to authenticate: %s - %s", errResp.Error, strings.ToLower(errResp.ErrorDescription))
		}

		// Try Airbyte API error format
		var apiErr struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
			return nil, fmt.Errorf("authentication failed: %s", apiErr.Message)
		}

		// Fallback to status code
		return nil, fmt.Errorf("authentication failed with status %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}
