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
	// OIDCProviderName is a string representation of the OIDC provider name.
	OIDCProviderName = "oidc"
)

// OIDCProvider represents an OIDC provider configuration
type OIDCProvider struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint,omitempty"`
	JwksURI               string `json:"jwks_uri,omitempty"`
	ClientID              string `json:"client_id,omitempty"`

	// Provider implementation fields
	httpClient http.HTTPDoer
	store      CredentialsStore
	mu         sync.Mutex
}

// RefreshToken implements Provider interface for OIDC
func (p *OIDCProvider) RefreshToken(ctx context.Context, refreshToken string, httpDoer http.HTTPDoer) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", p.ClientID)
	data.Set("refresh_token", refreshToken)

	return doTokenRequest(ctx, p.TokenEndpoint, data, httpDoer)
}

// TokenEndpoint implements Provider interface for OIDC
func (p *OIDCProvider) GetTokenEndpoint() string {
	return p.TokenEndpoint
}

// AuthEndpointHandler implements Provider interface for OIDC
func (p *OIDCProvider) AuthEndpointHandler() func(clientID, redirectURI, state, codeChallenge string) string {
	return func(clientID, redirectURI, state, codeChallenge string) string {
		params := url.Values{}
		params.Set("client_id", clientID)
		params.Set("response_type", "code")
		params.Set("redirect_uri", redirectURI)
		params.Set("state", state)
		params.Set("code_challenge", codeChallenge)
		params.Set("code_challenge_method", "S256")
		params.Set("scope", "openid profile email offline_access")
		return fmt.Sprintf("%s?%s", p.AuthorizationEndpoint, params.Encode())
	}
}

// NewOIDCProvider creates a new OIDC provider that implements the Provider interface
func NewOIDCProvider(httpClient http.HTTPDoer, store CredentialsStore) *OIDCProvider {
	return &OIDCProvider{
		httpClient: httpClient,
		store:      store,
	}
}

// Do implements http.HTTPDoer interface with OIDC authentication
func (p *OIDCProvider) Do(req *stdhttp.Request) (*stdhttp.Response, error) {
	creds, err := p.ensureValidToken(req.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("%s %s", creds.TokenType, creds.AccessToken))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Handle 401 with token refresh
	if resp.StatusCode == stdhttp.StatusUnauthorized {
		_ = resp.Body.Close()

		p.mu.Lock()
		// Mark current token as expired to force refresh
		_ = p.store.Save(&Credentials{ExpiresAt: time.Now().Add(-defaultTokenExpiry)})
		newCreds, refreshErr := p.ensureValidTokenLocked(req.Context())
		p.mu.Unlock()

		if refreshErr != nil {
			return nil, fmt.Errorf("failed to refresh token after 401: %w", refreshErr)
		}

		req.Header.Set("Authorization", fmt.Sprintf("%s %s", newCreds.TokenType, newCreds.AccessToken))
		return p.httpClient.Do(req)
	}

	return resp, nil
}

// SetCredentialsStore sets the credentials store for the provider
func (p *OIDCProvider) SetCredentialsStore(store CredentialsStore) {
	p.store = store
}

// Load implements CredentialsStore interface
func (p *OIDCProvider) Load() (*Credentials, error) {
	return p.store.Load()
}

// Save implements CredentialsStore interface
func (p *OIDCProvider) Save(creds *Credentials) error {
	return p.store.Save(creds)
}

func (p *OIDCProvider) ensureValidToken(ctx context.Context) (*Credentials, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ensureValidTokenLocked(ctx)
}

func (p *OIDCProvider) ensureValidTokenLocked(ctx context.Context) (*Credentials, error) {
	creds, err := p.store.Load()
	if err != nil {
		creds = nil
	}

	if creds == nil || creds.IsExpired() {
		if creds != nil && creds.RefreshToken != "" {
			tokenResp, refreshErr := p.RefreshToken(ctx, creds.RefreshToken, p.httpClient)
			if refreshErr != nil {
				return nil, fmt.Errorf("failed to refresh token: %w", refreshErr)
			}

			expiresAt := time.Now().Add(defaultTokenExpiry)
			if tokenResp.ExpiresIn > 0 {
				expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
			}
			creds = &Credentials{
				AccessToken:  tokenResp.AccessToken,
				RefreshToken: tokenResp.RefreshToken,
				TokenType:    tokenResp.TokenType,
				ExpiresAt:    expiresAt,
			}
			if creds.TokenType == "" {
				creds.TokenType = "Bearer"
			}
			_ = p.store.Save(creds)
			return creds, nil
		}
		return nil, fmt.Errorf("no valid credentials available")
	}

	return creds, nil
}
