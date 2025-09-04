//go:generate go run go.uber.org/mock/mockgen -destination=mock/config_provider.go -package=mock . ConfigProvider

package airbox

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/airbytehq/abctl/internal/abctl"
	"github.com/airbytehq/abctl/internal/auth"
	"github.com/airbytehq/abctl/internal/http"
)

// ConfigProvider interface for testable config operations
type ConfigProvider interface {
	Load() (*Config, error)
	Save(config *Config) error
	GetPath() string
}

// FileConfigProvider implements ConfigProvider using filesystem
type FileConfigProvider struct{}

// DefaultConfigProvider is the default filesystem-based provider
var DefaultConfigProvider ConfigProvider = &FileConfigProvider{}

// Config represents the airbox configuration file structure
type Config struct {
	CurrentContext string         `json:"current-context" yaml:"current-context"`
	Credentials    *Credentials   `json:"user,omitempty" yaml:"user,omitempty"`
	Contexts       []NamedContext `json:"contexts" yaml:"contexts"`
}

// NamedContext represents a named context entry
type NamedContext struct {
	Name    string  `json:"name" yaml:"name"`
	Context Context `json:"context" yaml:"context"`
}

// Context represents a single airbox context (org/workspace combination)
type Context struct {
	AirbyteAPIHost string `json:"airbyteApiHost" yaml:"airbyteApiHost"`
	AirbyteURL     string `json:"airbyteUrl" yaml:"airbyteUrl"`
	AirbyteAuthURL string `json:"airbyteAuthUrl" yaml:"airbyteAuthUrl"`
	OrganizationID string `json:"organizationId" yaml:"organizationId"`
	OIDCClientID   string `json:"oidcClientId" yaml:"oidcClientId"`
	Edition        string `json:"edition" yaml:"edition"`
}

// Credentials represents the authenticated user credentials
type Credentials struct {
	AccessToken  string    `json:"accessToken" yaml:"accessToken"`
	RefreshToken string    `json:"refreshToken" yaml:"refreshToken"`
	TokenType    string    `json:"tokenType" yaml:"tokenType"`
	ExpiresAt    time.Time `json:"expiresAt" yaml:"expiresAt"`
}

// ToAuthCredentials converts airbox.Credentials to auth.Credentials
func (c *Credentials) ToAuthCredentials() (*auth.Credentials, error) {
	if c == nil {
		return nil, fmt.Errorf("cannot convert nil credentials")
	}
	return &auth.Credentials{
		AccessToken:  c.AccessToken,
		RefreshToken: c.RefreshToken,
		TokenType:    c.TokenType,
		ExpiresAt:    c.ExpiresAt,
	}, nil
}

// FromAuthCredentials converts auth.Credentials to airbox.Credentials
func FromAuthCredentials(creds *auth.Credentials) (*Credentials, error) {
	if creds == nil {
		return nil, fmt.Errorf("cannot convert nil auth credentials")
	}
	return &Credentials{
		AccessToken:  creds.AccessToken,
		RefreshToken: creds.RefreshToken,
		TokenType:    creds.TokenType,
		ExpiresAt:    creds.ExpiresAt,
	}, nil
}

// DefaultConfigPath returns the default path for airbox config
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".airbyte", "airbox", "config")
	}
	return filepath.Join(home, ".airbyte", "airbox", "config")
}

// GetConfigPath returns the config path from environment or default
func GetConfigPath() string {
	if path := os.Getenv("AIRBOXCONFIG"); path != "" {
		return path
	}
	return DefaultConfigPath()
}

// FileConfigProvider methods
func (p *FileConfigProvider) Load() (*Config, error) {
	configPath := p.GetPath()

	// If file doesn't exist, return empty config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{
			Contexts: make([]NamedContext, 0),
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func (p *FileConfigProvider) Save(config *Config) error {
	configPath := p.GetPath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (p *FileConfigProvider) GetPath() string {
	if path := os.Getenv("AIRBOXCONFIG"); path != "" {
		return path
	}
	return DefaultConfigPath()
}

// Convenience functions that use the default provider
func LoadConfig() (*Config, error) {
	return DefaultConfigProvider.Load()
}

// GetCurrentContext returns the current context
func (c *Config) GetCurrentContext() (*Context, error) {
	if c.CurrentContext == "" {
		return nil, fmt.Errorf("no current context set")
	}

	for _, namedContext := range c.Contexts {
		if namedContext.Name == c.CurrentContext {
			return &namedContext.Context, nil
		}
	}

	return nil, fmt.Errorf("current context %q not found", c.CurrentContext)
}

// SetCurrentContext sets the current context
func (c *Config) SetCurrentContext(name string) error {
	// Verify context exists
	for _, namedContext := range c.Contexts {
		if namedContext.Name == name {
			c.CurrentContext = name
			return nil
		}
	}

	return fmt.Errorf("context %q not found", name)
}

// AddContext adds or updates a context
func (c *Config) AddContext(name string, context Context) {
	// Check if context already exists
	for i, namedContext := range c.Contexts {
		if namedContext.Name == name {
			c.Contexts[i].Context = context
			return
		}
	}

	// Add new context
	c.Contexts = append(c.Contexts, NamedContext{
		Name:    name,
		Context: context,
	})

	// Set as current if it's the first context
	if c.CurrentContext == "" {
		c.CurrentContext = name
	}
}

// RemoveContext removes a context
func (c *Config) RemoveContext(name string) error {
	for i, namedContext := range c.Contexts {
		if namedContext.Name == name {
			c.Contexts = append(c.Contexts[:i], c.Contexts[i+1:]...)

			// If this was the current context, clear it
			if c.CurrentContext == name {
				if len(c.Contexts) > 0 {
					c.CurrentContext = c.Contexts[0].Name
				} else {
					c.CurrentContext = ""
				}
			}
			return nil
		}
	}

	return fmt.Errorf("context %q not found", name)
}

// ToAbctlConfig converts the current context to abctl.Config format
func (c *Config) ToAbctlConfig() (*abctl.Config, error) {
	currentCtx, err := c.GetCurrentContext()
	if err != nil {
		return nil, err
	}

	return &abctl.Config{
		AirbyteAPIHost: currentCtx.AirbyteAPIHost,
		AirbyteURL:     currentCtx.AirbyteURL,
		AirbyteAuthURL: currentCtx.AirbyteAuthURL,
		OrganizationID: currentCtx.OrganizationID,
		OIDCClientID:   currentCtx.OIDCClientID,
		Edition:        currentCtx.Edition,
	}, nil
}

// GetCredentials returns the current user credentials
func (c *Config) GetCredentials() (*Credentials, error) {
	if c.Credentials == nil {
		return nil, fmt.Errorf("no user credentials found")
	}
	return c.Credentials, nil
}

// IsAuthenticated checks if user has valid credentials
func (c *Config) IsAuthenticated() error {
	if c.Credentials == nil || c.Credentials.AccessToken == "" {
		return fmt.Errorf("not authenticated - please run 'airbox auth login' first")
	}
	authCreds, err := c.Credentials.ToAuthCredentials()
	if err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}
	if authCreds.IsExpired() {
		return fmt.Errorf("authentication token has expired - please run 'airbox auth login' first")
	}
	return nil
}

// SetCredentials updates the user credentials in the config (does not save)
func (c *Config) SetCredentials(creds *Credentials) {
	c.Credentials = creds
}

// CreateCredentialsUpdateHook creates a standard update hook for saving refreshed credentials
func CreateCredentialsUpdateHook(cfg ConfigProvider) auth.CredentialsUpdateHook {
	return func(creds *auth.Credentials) error {
		abCfg, err := cfg.Load()
		if err != nil {
			return err
		}
		abCfg.Credentials, err = FromAuthCredentials(creds)
		if err != nil {
			return fmt.Errorf("failed to convert credentials: %w", err)
		}
		return cfg.Save(abCfg)
	}
}

// CreateHTTPClient creates an authenticated HTTP client from config
func CreateHTTPClient(ctx context.Context, cfg ConfigProvider, httpDoer http.HTTPDoer) (http.HTTPDoer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cfg cannot be nil")
	}

	if httpDoer == nil {
		return nil, fmt.Errorf("httpDoer cannot be nil")
	}

	abCfg, err := cfg.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	config, err := abCfg.ToAbctlConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get current context: %w", err)
	}

	creds, err := abCfg.GetCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	authCreds := &auth.Credentials{
		AccessToken:  creds.AccessToken,
		RefreshToken: creds.RefreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    creds.ExpiresAt,
	}

	authProvider, err := auth.DiscoverProviderWithClient(ctx, config.AirbyteAuthURL, httpDoer)
	if err != nil {
		return nil, fmt.Errorf("failed to discover auth provider: %w", err)
	}

	authClient := auth.NewClient(authProvider, config.OIDCClientID, authCreds, httpDoer, CreateCredentialsUpdateHook(cfg))

	return http.NewClient(config.AirbyteAPIHost, authClient)
}
