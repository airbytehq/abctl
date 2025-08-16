package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ProviderConfig holds OIDC provider configuration
type ProviderConfig struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JwksURI               string `json:"jwks_uri"`
	EndSessionEndpoint    string `json:"end_session_endpoint,omitempty"`
}

// DiscoverProvider fetches OIDC provider configuration from well-known endpoint
func DiscoverProvider(ctx context.Context, issuerURL string) (*ProviderConfig, error) {
	wellKnownURL := fmt.Sprintf("%s/.well-known/openid-configuration", issuerURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", wellKnownURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch provider config: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch provider config: status %d", resp.StatusCode)
	}
	
	var config ProviderConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode provider config: %w", err)
	}
	
	return &config, nil
}