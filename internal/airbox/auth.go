package airbox

import (
	"encoding/json"
	"fmt"

	"github.com/airbytehq/abctl/internal/auth"
	"gopkg.in/yaml.v3"
)

// AuthProvider interface for authentication configuration
type AuthProvider interface {
	Type() string
	Validate() error
}

// OIDC configuration for OIDC providers
type OIDC struct {
	AuthURL  string `yaml:"authUrl" json:"authUrl"`
	ClientID string `yaml:"clientId" json:"clientId"`
}

// Type returns the authentication type
func (o *OIDC) Type() string { return auth.OIDCProviderName }

// Validate checks OIDC configuration is complete
func (o *OIDC) Validate() error {
	if o.AuthURL == "" {
		return fmt.Errorf("OIDC auth missing required 'authUrl' field")
	}
	if o.ClientID == "" {
		return fmt.Errorf("OIDC auth missing required 'clientId' field")
	}
	return nil
}

// OAuth2 configuration for OAuth2 providers
type OAuth2 struct {
	ClientID     string `yaml:"clientId" json:"clientId"`
	ClientSecret string `yaml:"clientSecret" json:"clientSecret"`
}

// Type returns the authentication type
func (o *OAuth2) Type() string { return auth.OAuth2ProviderName }

// Validate checks OAuth2 configuration is complete
func (o *OAuth2) Validate() error {
	if o.ClientID == "" {
		return fmt.Errorf("OAuth2 auth missing required 'clientId' field")
	}
	if o.ClientSecret == "" {
		return fmt.Errorf("OAuth2 auth missing required 'clientSecret' field")
	}
	return nil
}

// NewOIDC creates a new OIDC auth configuration
func NewOIDC(authURL, clientID string) *OIDC {
	return &OIDC{
		AuthURL:  authURL,
		ClientID: clientID,
	}
}

// NewOAuth2 creates a new OAuth2 auth configuration
func NewOAuth2(clientID, clientSecret string) *OAuth2 {
	return &OAuth2{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
}

// Auth wraps AuthProvider for marshaling/unmarshaling
type Auth struct {
	provider AuthProvider
}

// NewAuthWithOIDC creates Auth with OIDC provider
func NewAuthWithOIDC(authURL, clientID string) Auth {
	return Auth{provider: NewOIDC(authURL, clientID)}
}

// NewAuthWithOAuth2 creates Auth with OAuth2 provider
func NewAuthWithOAuth2(clientID, clientSecret string) Auth {
	return Auth{provider: NewOAuth2(clientID, clientSecret)}
}

// GetProvider returns the underlying AuthProvider
func (w *Auth) GetProvider() AuthProvider {
	return w.provider
}

// GetOAuth2Provider returns the provider as an OAuth2Provider or an error
func (w *Auth) GetOAuth2Provider() (*OAuth2, error) {
	if w.provider == nil {
		return nil, fmt.Errorf("auth provider is nil")
	}
	if p, ok := w.provider.(*OAuth2); ok {
		return p, nil
	}
	return nil, fmt.Errorf("auth provider is not OAuth2, got %s", w.provider.Type())
}

// GetOIDCProvider returns the provider as an OIDCProvider or an error
func (w *Auth) GetOIDCProvider() (*OIDC, error) {
	if w.provider == nil {
		return nil, fmt.Errorf("auth provider is nil")
	}
	if p, ok := w.provider.(*OIDC); ok {
		return p, nil
	}
	return nil, fmt.Errorf("auth provider is not OIDC, got %s", w.provider.Type())
}

// Validate ensures the Auth struct is properly configured
// Validate checks authentication configuration is valid
func (w *Auth) Validate() error {
	if w.provider == nil {
		return fmt.Errorf("provider is nil")
	}

	return w.provider.Validate()
}

// MarshalYAML implements custom YAML marshaling
// MarshalYAML serializes Auth to YAML with type discrimination
func (w Auth) MarshalYAML() (interface{}, error) {
	if w.provider == nil {
		return nil, nil
	}

	if err := w.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid Auth: %w", err)
	}
	result := map[string]interface{}{
		"type": w.provider.Type(),
	}

	switch auth := w.provider.(type) {
	case *OIDC:
		result["authUrl"] = auth.AuthURL
		result["clientId"] = auth.ClientID
	case *OAuth2:
		result["clientId"] = auth.ClientID
		result["clientSecret"] = auth.ClientSecret
	default:
		return nil, fmt.Errorf("unknown auth type: %T", auth)
	}

	return result, nil
}

// UnmarshalYAML implements custom YAML unmarshaling
// UnmarshalYAML deserializes YAML to Auth with type discrimination
func (w *Auth) UnmarshalYAML(value *yaml.Node) error {
	var raw map[string]interface{}
	if err := value.Decode(&raw); err != nil {
		return fmt.Errorf("failed to decode auth config: %w", err)
	}

	typeVal, ok := raw["type"].(string)
	if !ok {
		return fmt.Errorf("auth config missing required 'type' field")
	}

	switch typeVal {
	case auth.OIDCProviderName:
		authURL, ok := raw["authUrl"].(string)
		if !ok || authURL == "" {
			return fmt.Errorf("OIDC auth missing required 'authUrl' field")
		}

		clientID, ok := raw["clientId"].(string)
		if !ok || clientID == "" {
			return fmt.Errorf("OIDC auth missing required 'clientId' field")
		}

		w.provider = &OIDC{
			AuthURL:  authURL,
			ClientID: clientID,
		}

	case auth.OAuth2ProviderName:
		clientID, ok := raw["clientId"].(string)
		if !ok || clientID == "" {
			return fmt.Errorf("OAuth2 auth missing required 'clientId' field")
		}

		clientSecret, ok := raw["clientSecret"].(string)
		if !ok || clientSecret == "" {
			return fmt.Errorf("OAuth2 auth missing required 'clientSecret' field")
		}

		w.provider = &OAuth2{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		}

	default:
		return fmt.Errorf("unknown auth type: %s", typeVal)
	}

	return w.provider.Validate()
}

// MarshalJSON serializes Auth to JSON with type discrimination
func (w *Auth) MarshalJSON() ([]byte, error) {
	if w.provider == nil {
		return json.Marshal(nil)
	}

	if err := w.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid Auth: %w", err)
	}

	result := map[string]interface{}{
		"type": w.provider.Type(),
	}

	switch auth := w.provider.(type) {
	case *OIDC:
		result["authUrl"] = auth.AuthURL
		result["clientId"] = auth.ClientID
	case *OAuth2:
		result["clientId"] = auth.ClientID
		result["clientSecret"] = auth.ClientSecret
	default:
		return nil, fmt.Errorf("unknown auth type: %T", auth)
	}

	return json.Marshal(result)
}

// UnmarshalJSON deserializes JSON to Auth with type discrimination
func (w *Auth) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		w.provider = nil
		return nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to decode auth config: %w", err)
	}

	typeVal, ok := raw["type"].(string)
	if !ok {
		return fmt.Errorf("auth config missing required 'type' field")
	}

	switch typeVal {
	case auth.OIDCProviderName:
		authURL, ok := raw["authUrl"].(string)
		if !ok || authURL == "" {
			return fmt.Errorf("OIDC auth missing required 'authUrl' field")
		}

		clientID, ok := raw["clientId"].(string)
		if !ok || clientID == "" {
			return fmt.Errorf("OIDC auth missing required 'clientId' field")
		}

		w.provider = &OIDC{
			AuthURL:  authURL,
			ClientID: clientID,
		}

	case auth.OAuth2ProviderName:
		clientID, ok := raw["clientId"].(string)
		if !ok || clientID == "" {
			return fmt.Errorf("OAuth2 auth missing required 'clientId' field")
		}

		clientSecret, ok := raw["clientSecret"].(string)
		if !ok || clientSecret == "" {
			return fmt.Errorf("OAuth2 auth missing required 'clientSecret' field")
		}

		w.provider = &OAuth2{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		}

	default:
		return fmt.Errorf("unknown auth type: %s", typeVal)
	}

	return w.provider.Validate()
}
