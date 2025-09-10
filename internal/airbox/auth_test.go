package airbox

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestOIDC_Type(t *testing.T) {
	oidc := &OIDC{AuthURL: "https://auth.example.com", ClientID: "test-client"}
	assert.Equal(t, "oidc", oidc.Type())
}

func TestOIDC_Validate(t *testing.T) {
	tests := []struct {
		name          string
		oidc          *OIDC
		expectedError string
	}{
		{
			name:          "valid OIDC",
			oidc:          &OIDC{AuthURL: "https://auth.example.com", ClientID: "test-client"},
			expectedError: "",
		},
		{
			name:          "missing AuthURL",
			oidc:          &OIDC{ClientID: "test-client"},
			expectedError: "OIDC auth missing required 'authUrl' field",
		},
		{
			name:          "missing ClientID",
			oidc:          &OIDC{AuthURL: "https://auth.example.com"},
			expectedError: "OIDC auth missing required 'clientId' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.oidc.Validate()
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOAuth2_Type(t *testing.T) {
	oauth := &OAuth2{ClientID: "client", ClientSecret: "secret"}
	assert.Equal(t, "oauth2", oauth.Type())
}

func TestNewOIDC(t *testing.T) {
	oidc := NewOIDC("https://auth.example.com", "test-client")
	assert.Equal(t, "https://auth.example.com", oidc.AuthURL)
	assert.Equal(t, "test-client", oidc.ClientID)
}

func TestNewOAuth2(t *testing.T) {
	oauth := NewOAuth2("client-id", "client-secret")
	assert.Equal(t, "client-id", oauth.ClientID)
	assert.Equal(t, "client-secret", oauth.ClientSecret)
}

func TestNewAuthWithOIDC(t *testing.T) {
	auth := NewAuthWithOIDC("https://auth.example.com", "test-client")
	provider := auth.GetProvider()
	require.NotNil(t, provider)

	oidc, ok := provider.(*OIDC)
	require.True(t, ok)
	assert.Equal(t, "https://auth.example.com", oidc.AuthURL)
	assert.Equal(t, "test-client", oidc.ClientID)
}

func TestNewAuthWithOAuth2(t *testing.T) {
	auth := NewAuthWithOAuth2("client-id", "client-secret")
	provider := auth.GetProvider()
	require.NotNil(t, provider)

	oauth, ok := provider.(*OAuth2)
	require.True(t, ok)
	assert.Equal(t, "client-id", oauth.ClientID)
	assert.Equal(t, "client-secret", oauth.ClientSecret)
}

func TestAuth_GetProvider(t *testing.T) {
	tests := []struct {
		name     string
		auth     Auth
		expected AuthProvider
	}{
		{
			name:     "OIDC provider",
			auth:     NewAuthWithOIDC("https://auth.example.com", "test-client"),
			expected: NewOIDC("https://auth.example.com", "test-client"),
		},
		{
			name:     "OAuth2 provider",
			auth:     NewAuthWithOAuth2("client-id", "client-secret"),
			expected: NewOAuth2("client-id", "client-secret"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := tt.auth.GetProvider()
			if tt.expected == nil {
				assert.Nil(t, provider)
			} else {
				assert.Equal(t, tt.expected, provider)
			}
		})
	}
}

func TestAuth_Validate(t *testing.T) {
	tests := []struct {
		name          string
		auth          Auth
		expectedError string
	}{
		{
			name:          "valid OAuth2",
			auth:          NewAuthWithOAuth2("client-id", "client-secret"),
			expectedError: "",
		},
		{
			name:          "valid OIDC",
			auth:          NewAuthWithOIDC("https://auth.example.com", "test-client"),
			expectedError: "",
		},
		{
			name:          "nil provider",
			auth:          Auth{},
			expectedError: "provider is nil",
		},
		{
			name:          "invalid OAuth2",
			auth:          NewAuthWithOAuth2("", "secret"),
			expectedError: "OAuth2 auth missing required 'clientId' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.auth.Validate()
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuth_MarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		auth     Auth
		expected map[string]interface{}
	}{
		{
			name: "OIDC auth",
			auth: NewAuthWithOIDC("https://auth.example.com", "test-client"),
			expected: map[string]interface{}{
				"type":     "oidc",
				"authUrl":  "https://auth.example.com",
				"clientId": "test-client",
			},
		},
		{
			name: "OAuth2 auth",
			auth: NewAuthWithOAuth2("client-id", "client-secret"),
			expected: map[string]interface{}{
				"type":         "oauth2",
				"clientId":     "client-id",
				"clientSecret": "client-secret",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.auth.MarshalYAML()
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuth_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name          string
		yamlData      string
		expectedAuth  Auth
		expectedError string
	}{
		{
			name: "OIDC auth",
			yamlData: `
type: oidc
authUrl: https://auth.example.com
clientId: test-client
`,
			expectedAuth: NewAuthWithOIDC("https://auth.example.com", "test-client"),
		},
		{
			name: "OAuth2 auth",
			yamlData: `
type: oauth2
clientId: client-id
clientSecret: client-secret
`,
			expectedAuth: NewAuthWithOAuth2("client-id", "client-secret"),
		},
		{
			name: "missing type",
			yamlData: `
authUrl: https://auth.example.com
clientId: test-client
`,
			expectedError: "auth config missing required 'type' field",
		},
		{
			name: "unknown type",
			yamlData: `
type: unknown
`,
			expectedError: "unknown auth type: unknown",
		},
		{
			name: "OIDC missing authUrl",
			yamlData: `
type: oidc
clientId: test-client
`,
			expectedError: "OIDC auth missing required 'authUrl' field",
		},
		{
			name: "OAuth2 missing clientId",
			yamlData: `
type: oauth2
clientSecret: secret
`,
			expectedError: "OAuth2 auth missing required 'clientId' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var auth Auth
			err := yaml.Unmarshal([]byte(tt.yamlData), &auth)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedAuth.GetProvider(), auth.GetProvider())
			}
		})
	}
}

func TestAuth_MarshalJSON(t *testing.T) {
	tests := []struct {
		name         string
		auth         Auth
		expectedJSON string
	}{
		{
			name:         "OIDC auth",
			auth:         NewAuthWithOIDC("https://auth.example.com", "test-client"),
			expectedJSON: `{"authUrl":"https://auth.example.com","clientId":"test-client","type":"oidc"}`,
		},
		{
			name:         "OAuth2 auth",
			auth:         NewAuthWithOAuth2("client-id", "client-secret"),
			expectedJSON: `{"clientId":"client-id","clientSecret":"client-secret","type":"oauth2"}`,
		},
		{
			name:         "nil provider",
			auth:         Auth{},
			expectedJSON: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(&tt.auth)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expectedJSON, string(data))
		})
	}
}

func TestAuth_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name          string
		jsonData      string
		expectedAuth  Auth
		expectedError string
	}{
		{
			name:         "OIDC auth",
			jsonData:     `{"type":"oidc","authUrl":"https://auth.example.com","clientId":"test-client"}`,
			expectedAuth: NewAuthWithOIDC("https://auth.example.com", "test-client"),
		},
		{
			name:         "OAuth2 auth",
			jsonData:     `{"type":"oauth2","clientId":"client-id","clientSecret":"client-secret"}`,
			expectedAuth: NewAuthWithOAuth2("client-id", "client-secret"),
		},
		{
			name:         "null",
			jsonData:     "null",
			expectedAuth: Auth{},
		},
		{
			name:          "missing type",
			jsonData:      `{"authUrl":"https://auth.example.com","clientId":"test-client"}`,
			expectedError: "auth config missing required 'type' field",
		},
		{
			name:          "unknown type",
			jsonData:      `{"type":"unknown"}`,
			expectedError: "unknown auth type: unknown",
		},
		{
			name:          "OIDC missing clientId",
			jsonData:      `{"type":"oidc","authUrl":"https://auth.example.com"}`,
			expectedError: "OIDC auth missing required 'clientId' field",
		},
		{
			name:          "OAuth2 missing clientSecret",
			jsonData:      `{"type":"oauth2","clientId":"client-id"}`,
			expectedError: "OAuth2 auth missing required 'clientSecret' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var auth Auth
			err := json.Unmarshal([]byte(tt.jsonData), &auth)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedAuth.GetProvider(), auth.GetProvider())
			}
		})
	}
}

func TestAuth_MarshalJSON_InvalidAuth(t *testing.T) {
	// Test marshaling with invalid auth
	auth := NewAuthWithOAuth2("", "secret") // Invalid - missing client ID
	_, err := auth.MarshalJSON()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot marshal invalid Auth")
}

func TestAuth_MarshalYAML_InvalidAuth(t *testing.T) {
	// Test marshaling with invalid auth
	auth := NewAuthWithOIDC("", "client") // Invalid - missing auth URL
	_, err := auth.MarshalYAML()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot marshal invalid Auth")
}

func TestAuth_GetOAuth2Provider(t *testing.T) {
	tests := []struct {
		name          string
		auth          Auth
		expectedError string
	}{
		{
			name: "success OAuth2",
			auth: NewAuthWithOAuth2("client-id", "client-secret"),
		},
		{
			name:          "error with OIDC provider",
			auth:          NewAuthWithOIDC("https://example.com", "client-id"),
			expectedError: "auth provider is not OAuth2, got oidc",
		},
		{
			name:          "error with nil provider",
			auth:          Auth{},
			expectedError: "auth provider is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := tt.auth.GetOAuth2Provider()
			
			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, "client-id", provider.ClientID)
				assert.Equal(t, "client-secret", provider.ClientSecret)
			}
		})
	}
}

func TestAuth_GetOIDCProvider(t *testing.T) {
	tests := []struct {
		name          string
		auth          Auth
		expectedError string
	}{
		{
			name: "success OIDC",
			auth: NewAuthWithOIDC("https://example.com", "client-id"),
		},
		{
			name:          "error with OAuth2 provider",
			auth:          NewAuthWithOAuth2("client-id", "client-secret"),
			expectedError: "auth provider is not OIDC, got oauth2",
		},
		{
			name:          "error with nil provider",
			auth:          Auth{},
			expectedError: "auth provider is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := tt.auth.GetOIDCProvider()
			
			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, "https://example.com", provider.AuthURL)
				assert.Equal(t, "client-id", provider.ClientID)
			}
		})
	}
}
