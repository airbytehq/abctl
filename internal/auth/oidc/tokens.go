package oidc

import (
	"encoding/json"
	"time"
)

// TokenResponse represents OAuth2 token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// Credentials holds authentication credentials
type Credentials struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	BaseURL      string    `json:"base_url"`
	OIDCServer   string    `json:"oidc_server"`
}

// IsExpired checks if the access token is expired
func (c *Credentials) IsExpired() bool {
	if c.ExpiresAt.IsZero() {
		return false // No expiry set, assume valid
	}
	// Check if expired with 1 minute buffer
	return time.Now().After(c.ExpiresAt.Add(-1 * time.Minute))
}

// NewCredentials creates credentials from token response
func NewCredentials(tokens *TokenResponse, baseURL, oidcServer string) *Credentials {
	creds := &Credentials{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		IDToken:      tokens.IDToken,
		BaseURL:      baseURL,
		OIDCServer:   oidcServer,
	}
	
	if tokens.ExpiresIn > 0 {
		creds.ExpiresAt = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	}
	
	return creds
}

// ToJSON serializes credentials to JSON
func (c *Credentials) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// FromJSON deserializes credentials from JSON
func FromJSON(data []byte) (*Credentials, error) {
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}