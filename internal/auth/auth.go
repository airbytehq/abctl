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

// TokenResponse represents an OAuth2 token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// Credentials stores authentication tokens and metadata
type Credentials struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// IsExpired checks if the access token has expired
func (c *Credentials) IsExpired() bool {
	// Consider expired 1 minute early to account for clock skew
	return time.Now().After(c.ExpiresAt.Add(-1 * time.Minute))
}

// ToJSON serializes credentials to JSON
func (c *Credentials) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// CredentialsFromJSON deserializes credentials from JSON
func CredentialsFromJSON(data []byte) (*Credentials, error) {
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

// Provider represents an OAuth2/OIDC provider configuration
type Provider struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint,omitempty"`
	JwksURI               string `json:"jwks_uri,omitempty"`
}

// DiscoverProvider fetches OIDC provider configuration from well-known endpoint
func DiscoverProvider(ctx context.Context, issuerURL string, client http.HTTPDoer) (*Provider, error) {
	return DiscoverProviderWithClient(ctx, issuerURL, client)
}

// DiscoverProviderWithClient fetches OIDC provider configuration using the provided HTTP client
func DiscoverProviderWithClient(ctx context.Context, issuerURL string, client http.HTTPDoer) (*Provider, error) {
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
	defer resp.Body.Close()

	if resp.StatusCode != stdhttp.StatusOK {
		return nil, fmt.Errorf("discovery failed with status %d", resp.StatusCode)
	}

	// Read response body for better error reporting
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read provider configuration response: %w", err)
	}

	var provider Provider
	if err := json.Unmarshal(body, &provider); err != nil {
		// Show first 500 characters of response to help debug
		preview := string(body)
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		return nil, fmt.Errorf("failed to decode provider configuration: %w\nReceived response: %s", err, preview)
	}

	return &provider, nil
}

// ExchangeCodeForTokens exchanges an authorization code for tokens
func ExchangeCodeForTokens(ctx context.Context, provider *Provider, clientID, code, redirectURI, codeVerifier string, client http.HTTPDoer) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", clientID)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}

	return doTokenRequest(ctx, provider.TokenEndpoint, data, client)
}

// RefreshAccessToken uses a refresh token to get a new access token
func RefreshAccessToken(ctx context.Context, provider *Provider, clientID, refreshToken string, client http.HTTPDoer) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", clientID)
	data.Set("refresh_token", refreshToken)

	return doTokenRequest(ctx, provider.TokenEndpoint, data, client)
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
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != stdhttp.StatusOK {
		var errResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("failed to decode error response for status code %d: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("token request failed: %s - %s", errResp.Error, errResp.ErrorDescription)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}
