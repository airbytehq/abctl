package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFlow(t *testing.T) {
	provider := &Provider{
		Issuer:                "https://auth.example.com",
		AuthorizationEndpoint: "https://auth.example.com/authorize",
		TokenEndpoint:         "https://auth.example.com/token",
	}

	flow := NewFlow(provider, "test-client", 8085)

	assert.NotNil(t, flow)
	assert.Equal(t, provider, flow.Provider)
	assert.Equal(t, "test-client", flow.ClientID)
	assert.Equal(t, 8085, flow.RedirectPort)
	assert.NotEmpty(t, flow.state)
	assert.NotEmpty(t, flow.codeVerifier)

	// Verify state is a valid UUID
	assert.Len(t, flow.state, 36) // UUID v4 format
	assert.Contains(t, flow.state, "-")

	// Verify code verifier is base64url encoded
	decoded, err := base64.RawURLEncoding.DecodeString(flow.codeVerifier)
	assert.NoError(t, err)
	assert.Len(t, decoded, 32) // Should be 32 bytes
}

func TestGenerateCodeVerifier(t *testing.T) {
	tests := []struct {
		name string
		runs int
	}{
		{
			name: "generates unique values",
			runs: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seen := make(map[string]bool)

			for i := 0; i < tt.runs; i++ {
				verifier := generateCodeVerifier()

				// Check it's not empty
				assert.NotEmpty(t, verifier)

				// Check it's unique
				assert.False(t, seen[verifier], "Generated duplicate verifier")
				seen[verifier] = true

				// Check it's valid base64url
				decoded, err := base64.RawURLEncoding.DecodeString(verifier)
				assert.NoError(t, err)
				assert.Len(t, decoded, 32)

				// Check it's URL-safe (no padding, no +, no /)
				assert.NotContains(t, verifier, "=")
				assert.NotContains(t, verifier, "+")
				assert.NotContains(t, verifier, "/")
			}
		})
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	tests := []struct {
		name     string
		verifier string
		want     string
	}{
		{
			name:     "empty verifier",
			verifier: "",
			want:     "47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU", // SHA256 of empty string
		},
		{
			name:     "known verifier",
			verifier: "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			want:     "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		},
		{
			name:     "another known verifier",
			verifier: "test-verifier",
			want:     calculateExpectedChallenge("test-verifier"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge := generateCodeChallenge(tt.verifier)
			assert.Equal(t, tt.want, challenge)

			// Verify it's valid base64url
			decoded, err := base64.RawURLEncoding.DecodeString(challenge)
			assert.NoError(t, err)
			assert.Len(t, decoded, 32) // SHA256 is 32 bytes

			// Verify it's URL-safe
			assert.NotContains(t, challenge, "=")
			assert.NotContains(t, challenge, "+")
			assert.NotContains(t, challenge, "/")
		})
	}
}

func TestFlow_BuildAuthURL(t *testing.T) {
	tests := []struct {
		name         string
		provider     *Provider
		clientID     string
		redirectPort int
		state        string
		codeVerifier string
		validate     func(*testing.T, string)
	}{
		{
			name: "builds correct auth URL with all parameters",
			provider: &Provider{
				AuthorizationEndpoint: "https://auth.example.com/authorize",
			},
			clientID:     "test-client",
			redirectPort: 8085,
			state:        "test-state",
			codeVerifier: "test-verifier",
			validate: func(t *testing.T, authURL string) {
				u, err := url.Parse(authURL)
				require.NoError(t, err)

				// Check base URL
				assert.Equal(t, "https", u.Scheme)
				assert.Equal(t, "auth.example.com", u.Host)
				assert.Equal(t, "/authorize", u.Path)

				// Check query parameters
				params := u.Query()
				assert.Equal(t, "test-client", params.Get("client_id"))
				assert.Equal(t, "code", params.Get("response_type"))
				assert.Equal(t, "http://localhost:8085/callback", params.Get("redirect_uri"))
				assert.Equal(t, "test-state", params.Get("state"))
				assert.Equal(t, "S256", params.Get("code_challenge_method"))
				assert.Equal(t, "openid profile email offline_access", params.Get("scope"))

				// Verify code challenge is correct
				expectedChallenge := calculateExpectedChallenge("test-verifier")
				assert.Equal(t, expectedChallenge, params.Get("code_challenge"))
			},
		},
		{
			name: "handles authorization endpoint with existing query params",
			provider: &Provider{
				AuthorizationEndpoint: "https://auth.example.com/authorize?tenant=default",
			},
			clientID:     "test-client",
			redirectPort: 8081,
			state:        "test-state",
			codeVerifier: "test-verifier",
			validate: func(t *testing.T, authURL string) {
				// The existing query param should be preserved (it's malformed but that's what happens)
				assert.Contains(t, authURL, "client_id=test-client")
				assert.Contains(t, authURL, "redirect_uri=http%3A%2F%2Flocalhost%3A8081%2Fcallback")
				// Original param is preserved in the malformed URL
				assert.Contains(t, authURL, "tenant=default")
			},
		},
		{
			name: "uses custom redirect port",
			provider: &Provider{
				AuthorizationEndpoint: "https://auth.example.com/authorize",
			},
			clientID:     "custom-client",
			redirectPort: 9999,
			state:        "custom-state",
			codeVerifier: "custom-verifier",
			validate: func(t *testing.T, authURL string) {
				assert.Contains(t, authURL, "redirect_uri=http%3A%2F%2Flocalhost%3A9999%2Fcallback")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flow := &Flow{
				Provider:     tt.provider,
				ClientID:     tt.clientID,
				RedirectPort: tt.redirectPort,
				state:        tt.state,
				codeVerifier: tt.codeVerifier,
			}

			authURL := flow.buildAuthURL()
			assert.NotEmpty(t, authURL)

			if tt.validate != nil {
				tt.validate(t, authURL)
			}
		})
	}
}

func TestPKCEFlow(t *testing.T) {
	t.Run("PKCE verifier and challenge relationship", func(t *testing.T) {
		// Generate a verifier
		verifier := generateCodeVerifier()
		assert.NotEmpty(t, verifier)

		// Generate challenge from verifier
		challenge := generateCodeChallenge(verifier)
		assert.NotEmpty(t, challenge)

		// Verify the challenge is the SHA256 of the verifier
		hash := sha256.Sum256([]byte(verifier))
		expectedChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
		assert.Equal(t, expectedChallenge, challenge)

		// Verify both are URL-safe
		for _, s := range []string{verifier, challenge} {
			assert.NotContains(t, s, "=", "Should not contain padding")
			assert.NotContains(t, s, "+", "Should not contain +")
			assert.NotContains(t, s, "/", "Should not contain /")
		}
	})
}

func TestAuthURLParameters(t *testing.T) {
	tests := []struct {
		name     string
		flow     *Flow
		expected map[string]string
	}{
		{
			name: "all required OAuth parameters present",
			flow: &Flow{
				Provider: &Provider{
					AuthorizationEndpoint: "https://auth.example.com/authorize",
				},
				ClientID:     "my-client",
				RedirectPort: 8085,
				state:        "state-123",
				codeVerifier: "verifier-456",
			},
			expected: map[string]string{
				"client_id":             "my-client",
				"response_type":         "code",
				"redirect_uri":          "http://localhost:8085/callback",
				"state":                 "state-123",
				"code_challenge_method": "S256",
				"scope":                 "openid profile email offline_access",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authURL := tt.flow.buildAuthURL()

			// Parse the URL
			u, err := url.Parse(authURL)
			require.NoError(t, err)

			// Check all expected parameters
			params := u.Query()
			for key, expectedValue := range tt.expected {
				assert.Equal(t, expectedValue, params.Get(key), "Parameter %s mismatch", key)
			}

			// Verify code_challenge is present and valid
			codeChallenge := params.Get("code_challenge")
			assert.NotEmpty(t, codeChallenge)

			// Verify it's a valid base64url string
			decoded, err := base64.RawURLEncoding.DecodeString(codeChallenge)
			assert.NoError(t, err)
			assert.Len(t, decoded, 32) // SHA256 output
		})
	}
}

func TestCallbackPath(t *testing.T) {
	// Verify the constant is what we expect
	assert.Equal(t, "/callback", CallbackPath)

	// Verify it's used in buildAuthURL
	flow := &Flow{
		Provider: &Provider{
			AuthorizationEndpoint: "https://auth.example.com/authorize",
		},
		ClientID:     "test",
		RedirectPort: 12345,
		state:        "test",
		codeVerifier: "test",
	}

	authURL := flow.buildAuthURL()
	assert.Contains(t, authURL, "redirect_uri=http%3A%2F%2Flocalhost%3A12345%2Fcallback")
	// The CallbackPath is URL-encoded in the query parameter
	assert.Contains(t, authURL, "%2Fcallback")
}

// Helper function to calculate expected challenge
func calculateExpectedChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func TestSuccessHTML(t *testing.T) {
	// Verify the HTML contains expected elements
	assert.Contains(t, successHTML, "<!DOCTYPE html>")
	assert.Contains(t, successHTML, "Authentication Successful")
	assert.Contains(t, successHTML, "You can close this window")
	assert.Contains(t, successHTML, "window.close()")

	// Verify it's valid HTML structure
	assert.Contains(t, successHTML, "<html>")
	assert.Contains(t, successHTML, "</html>")
	assert.Contains(t, successHTML, "<head>")
	assert.Contains(t, successHTML, "</head>")
	assert.Contains(t, successHTML, "<body>")
	assert.Contains(t, successHTML, "</body>")

	// Verify styling exists
	assert.Contains(t, successHTML, "<style>")
	assert.Contains(t, successHTML, "</style>")

	// Verify auto-close script
	assert.Contains(t, successHTML, "setTimeout")
	assert.Contains(t, successHTML, "3000") // 3 second delay
}

func TestDefaultConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{
			name:     "DefaultClientID",
			got:      DefaultClientID,
			expected: "abctl",
		},
		{
			name:     "CallbackPath",
			got:      CallbackPath,
			expected: "/callback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.got)
		})
	}
}

func TestURLEncoding(t *testing.T) {
	t.Run("redirect URI properly encoded in auth URL", func(t *testing.T) {
		flow := &Flow{
			Provider: &Provider{
				AuthorizationEndpoint: "https://auth.example.com/authorize",
			},
			ClientID:     "test",
			RedirectPort: 8085,
			state:        "test",
			codeVerifier: "test",
		}

		authURL := flow.buildAuthURL()

		// Check that special characters are properly encoded
		assert.Contains(t, authURL, "http%3A%2F%2Flocalhost")
		assert.NotContains(t, authURL, "http://localhost") // Should be encoded

		// Verify the URL can be parsed
		u, err := url.Parse(authURL)
		assert.NoError(t, err)

		// Verify redirect_uri decodes correctly
		redirectURI := u.Query().Get("redirect_uri")
		assert.Equal(t, "http://localhost:8085/callback", redirectURI)
	})

	t.Run("scope properly encoded", func(t *testing.T) {
		flow := &Flow{
			Provider: &Provider{
				AuthorizationEndpoint: "https://auth.example.com/authorize",
			},
			ClientID:     "test",
			RedirectPort: 8085,
			state:        "test",
			codeVerifier: "test",
		}

		authURL := flow.buildAuthURL()

		// Spaces in scope should be encoded as +
		assert.Contains(t, authURL, "scope=openid+profile+email+offline_access")

		// Verify it decodes correctly
		u, err := url.Parse(authURL)
		assert.NoError(t, err)
		scope := u.Query().Get("scope")
		assert.Equal(t, "openid profile email offline_access", scope)
	})
}
