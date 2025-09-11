package auth

import (
	"context"
	"fmt"
	stdhttp "net/http"
	"net/url"
	"sync"
	"time"

	"github.com/airbytehq/abctl/internal/http"
)

const (
	// OAuth2ProviderName is a string representation of the OAuth2 provider name.
	OAuth2ProviderName = "oauth2"
)

// OAuth2ClientCredentialsProvider represents an OAuth2 client credentials provider
type OAuth2ClientCredentialsProvider struct {
	TokenEndpoint string
	ClientID      string
	ClientSecret  string

	// Provider implementation fields
	httpClient http.HTTPDoer
	store      CredentialsStore
	mu         sync.Mutex
}

// NewOAuth2Provider creates an OAuth2 provider that implements the Provider interface
func NewOAuth2Provider(ctx context.Context, clientID, clientSecret string, httpClient http.HTTPDoer, store CredentialsStore) (*OAuth2ClientCredentialsProvider, error) {
	provider := &OAuth2ClientCredentialsProvider{
		TokenEndpoint: "/v1/applications/token",
		ClientID:      clientID,
		ClientSecret:  clientSecret,
		httpClient:    httpClient,
		store:         store,
	}

	// Fetch initial tokens to validate credentials
	_, err := provider.ensureValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with OAuth2 credentials: %w", err)
	}

	return provider, nil
}

// GetToken implements Provider interface for OAuth2 client credentials flow
func (p *OAuth2ClientCredentialsProvider) getToken(ctx context.Context, httpDoer http.HTTPDoer) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", p.ClientID)
	data.Set("client_secret", p.ClientSecret)

	return doTokenRequest(ctx, p.TokenEndpoint, data, httpDoer)
}

// RefreshToken implements Provider interface for OAuth2 client credentials
func (p *OAuth2ClientCredentialsProvider) RefreshToken(ctx context.Context, refreshToken string, httpDoer http.HTTPDoer) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", p.ClientID)
	data.Set("client_secret", p.ClientSecret)
	data.Set("refresh_token", refreshToken)

	return doTokenRequest(ctx, p.TokenEndpoint, data, httpDoer)
}

// GetTokenEndpoint implements Provider interface for OAuth2
func (p *OAuth2ClientCredentialsProvider) GetTokenEndpoint() string {
	return p.TokenEndpoint
}

// AuthEndpointHandler implements Provider interface for OAuth2 (not supported)
func (p *OAuth2ClientCredentialsProvider) AuthEndpointHandler() func(clientID, redirectURI, state, codeChallenge string) string {
	return nil
}

// Do implements http.HTTPDoer interface with OAuth2 authentication
func (p *OAuth2ClientCredentialsProvider) Do(req *stdhttp.Request) (*stdhttp.Response, error) {
	creds, err := p.ensureValidToken(req.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("%s %s", creds.TokenType, creds.AccessToken))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Handle 401 with fresh token (OAuth2 doesn't refresh, gets new token)
	if resp.StatusCode == stdhttp.StatusUnauthorized {
		_ = resp.Body.Close()

		p.mu.Lock()
		// Mark current token as expired to force getting a new one
		_ = p.store.Save(&Credentials{ExpiresAt: time.Now().Add(-defaultTokenExpiry)})
		newCreds, refreshErr := p.ensureValidTokenLocked(req.Context())
		p.mu.Unlock()

		if refreshErr != nil {
			return nil, fmt.Errorf("failed to get fresh token after 401: %w", refreshErr)
		}

		req.Header.Set("Authorization", fmt.Sprintf("%s %s", newCreds.TokenType, newCreds.AccessToken))
		return p.httpClient.Do(req)
	}

	return resp, nil
}

// Load implements CredentialsStore interface
func (p *OAuth2ClientCredentialsProvider) Load() (*Credentials, error) {
	return p.store.Load()
}

// Save implements CredentialsStore interface
func (p *OAuth2ClientCredentialsProvider) Save(creds *Credentials) error {
	return p.store.Save(creds)
}

func (p *OAuth2ClientCredentialsProvider) ensureValidToken(ctx context.Context) (*Credentials, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ensureValidTokenLocked(ctx)
}

func (p *OAuth2ClientCredentialsProvider) ensureValidTokenLocked(ctx context.Context) (*Credentials, error) {
	creds, err := p.store.Load()
	if err != nil {
		creds = nil
	}

	// OAuth2 client credentials: always get fresh token if expired
	if creds == nil || creds.IsExpired() {
		tokenResp, err := p.getToken(ctx, p.httpClient)
		if err != nil {
			return nil, fmt.Errorf("failed to get fresh token: %w", err)
		}

		expiresAt := time.Now().Add(defaultTokenExpiry)
		if tokenResp.ExpiresIn > 0 {
			expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		}

		tokenType := tokenResp.TokenType
		if tokenType == "" {
			tokenType = "Bearer"
		}

		creds = &Credentials{
			AccessToken:  tokenResp.AccessToken,
			RefreshToken: tokenResp.RefreshToken,
			TokenType:    tokenType,
			ExpiresAt:    expiresAt,
		}

		_ = p.store.Save(creds)
	}

	return creds, nil
}
