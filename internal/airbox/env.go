package airbox

import (
	"context"

	"github.com/airbytehq/abctl/internal/auth"
	"github.com/airbytehq/abctl/internal/http"
	"github.com/kelseyhightower/envconfig"
)

// OAuthEnvConfig holds environment-based configuration
type OAuthEnvConfig struct {
	ClientID     string `envconfig:"AIRBYTE_CLIENT_ID" required:"true"`
	ClientSecret string `envconfig:"AIRBYTE_CLIENT_SECRET" required:"true"`
}

// LoadOAuthEnvConfig loads configuration from environment variables
func LoadOAuthEnvConfig() (*OAuthEnvConfig, error) {
	var cfg OAuthEnvConfig
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ToAuthClient creates an authenticated HTTP client from stored credentials
func (c *OAuthEnvConfig) ToAuthClient(ctx context.Context, httpClient http.HTTPDoer, cfg ConfigStore) (auth.Provider, error) {
	// Create credentials store from config provider
	store := NewCredentialStoreAdapter(cfg)

	// Create OAuth2 provider that implements the Provider interface (HTTPDoer + CredentialsStore)
	// The provider IS the authenticated HTTP client and fetches initial tokens during creation
	provider, err := auth.NewOAuth2Provider(ctx, c.ClientID, c.ClientSecret, httpClient, store)
	if err != nil {
		return nil, err
	}

	return provider, nil
}
