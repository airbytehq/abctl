package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFlow(t *testing.T) {
	provider := &Provider{
		Issuer:                "https://auth.example.com",
		AuthorizationEndpoint: "https://auth.example.com/authorize",
		TokenEndpoint:         "https://auth.example.com/token",
	}

	flow := NewFlow(provider, "test-client", 8085, http.DefaultClient)

	assert.NotNil(t, flow)
	assert.Equal(t, provider, flow.provider)
	assert.Equal(t, "test-client", flow.clientID)
	assert.Equal(t, 8085, flow.redirectPort)
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
			name: "builds auth URL",
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
			name: "existing query params",
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
				provider:     tt.provider,
				clientID:     tt.clientID,
				redirectPort: tt.redirectPort,
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
			name: "OAuth parameters present",
			flow: &Flow{
				provider: &Provider{
					AuthorizationEndpoint: "https://auth.example.com/authorize",
				},
				clientID:     "my-client",
				redirectPort: 8085,
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
		provider: &Provider{
			AuthorizationEndpoint: "https://auth.example.com/authorize",
		},
		clientID:     "test",
		redirectPort: 12345,
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
			provider: &Provider{
				AuthorizationEndpoint: "https://auth.example.com/authorize",
			},
			clientID:     "test",
			redirectPort: 8085,
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
			provider: &Provider{
				AuthorizationEndpoint: "https://auth.example.com/authorize",
			},
			clientID:     "test",
			redirectPort: 8085,
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

func TestFlowGetAuthURL(t *testing.T) {
	provider := &Provider{
		AuthorizationEndpoint: "https://auth.example.com/authorize",
	}

	flow := NewFlow(provider, "test-client", 8085, nil)

	authURL := flow.GetAuthURL()
	assert.NotEmpty(t, authURL)
	assert.Contains(t, authURL, "https://auth.example.com/authorize")
	assert.Contains(t, authURL, "client_id=test-client")
	assert.Contains(t, authURL, "redirect_uri=http%3A%2F%2Flocalhost%3A8085%2Fcallback")
}

func TestWithStateGenerator(t *testing.T) {
	tests := []struct {
		name      string
		generator func() string
		expected  string
	}{
		{
			name:      "custom state",
			generator: func() string { return "custom-test-state" },
			expected:  "custom-test-state",
		},
		{
			name:      "fixed state",
			generator: func() string { return "fixed-state-123" },
			expected:  "fixed-state-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &Provider{
				AuthorizationEndpoint: "https://auth.example.com/authorize",
			}

			flow := NewFlow(provider, "test-client", 8085, nil, WithStateGenerator(tt.generator))

			assert.Equal(t, tt.expected, flow.state)

			// Verify it's used in auth URL
			authURL := flow.GetAuthURL()
			assert.Contains(t, authURL, "state="+tt.expected)
		})
	}
}


func TestFlowExchangeCode(t *testing.T) {
	tests := []struct {
		name                string
		code                string
		serverStatus        int
		serverBody          string
		expectedError       string
		expectedAccessToken string
		expectedRefreshToken string
		expectedTokenType   string
		expectedExpiresIn   int
	}{
		{
			name:                "success",
			code:                "valid-code",
			serverStatus:        http.StatusOK,
			serverBody: `{
				"access_token": "new-access-token",
				"refresh_token": "new-refresh-token",
				"token_type": "Bearer",
				"expires_in": 7200
			}`,
			expectedAccessToken:  "new-access-token",
			expectedRefreshToken: "new-refresh-token",
			expectedTokenType:    "Bearer",
			expectedExpiresIn:    7200,
		},
		{
			name:          "invalid response",
			code:          "bad-code",
			serverStatus:  http.StatusBadRequest,
			serverBody:    `{"error": "invalid_grant"}`,
			expectedError: "failed to exchange code for tokens",
		},
		{
			name:          "malformed json",
			code:          "code",
			serverStatus:  http.StatusOK,
			serverBody:    `{invalid json}`,
			expectedError: "invalid character",
		},
		{
			name:         "default expires in",
			code:         "valid-code",
			serverStatus: http.StatusOK,
			serverBody: `{
				"access_token": "token-no-expiry",
				"refresh_token": "refresh-no-expiry",
				"token_type": "Bearer"
			}`,
			expectedAccessToken:  "token-no-expiry",
			expectedRefreshToken: "refresh-no-expiry",
			expectedTokenType:    "Bearer",
			expectedExpiresIn:    3600, // Default 1 hour
		},
		{
			name:         "default token type",
			code:         "valid-code",
			serverStatus: http.StatusOK,
			serverBody: `{
				"access_token": "token-no-type",
				"refresh_token": "refresh-no-type",
				"expires_in": 3600
			}`,
			expectedAccessToken:  "token-no-type",
			expectedRefreshToken: "refresh-no-type",
			expectedTokenType:    "Bearer", // Should default to Bearer
			expectedExpiresIn:    3600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

				// Parse form
				err := r.ParseForm()
				require.NoError(t, err)

				assert.Equal(t, "authorization_code", r.Form.Get("grant_type"))
				assert.Equal(t, tt.code, r.Form.Get("code"))
				assert.Equal(t, "test-client", r.Form.Get("client_id"))
				assert.NotEmpty(t, r.Form.Get("code_verifier"))

				// Send response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.serverStatus)
				fmt.Fprint(w, tt.serverBody)
			}))
			defer server.Close()

			provider := &Provider{
				TokenEndpoint: server.URL,
			}

			flow := &Flow{
				provider:     provider,
				clientID:     "test-client",
				redirectPort: 8085,
				httpClient:   http.DefaultClient,
				state:        "test-state",
				codeVerifier: "test-verifier",
			}

			creds, err := flow.exchangeCode(context.Background(), tt.code)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, creds)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, creds)

				assert.Equal(t, tt.expectedAccessToken, creds.AccessToken)
				assert.Equal(t, tt.expectedRefreshToken, creds.RefreshToken)
				assert.Equal(t, tt.expectedTokenType, creds.TokenType)
				
				// Check expiry time
				expectedExpiry := time.Now().Add(time.Duration(tt.expectedExpiresIn) * time.Second)
				assert.WithinDuration(t, expectedExpiry, creds.ExpiresAt, 5*time.Second)
			}
		})
	}
}

func TestFlow_StartCallbackServer(t *testing.T) {
	tests := []struct {
		name          string
		port          int
		expectedError string
	}{
		{
			name: "success",
			port: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &Provider{
				AuthorizationEndpoint: "https://auth.example.com/authorize",
				TokenEndpoint:        "https://auth.example.com/token",
			}

			flow := NewFlow(provider, "test-client", tt.port, http.DefaultClient)

			listener, err := flow.StartCallbackServer()
			
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, listener)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, listener)
				
				// Verify listener is working
				addr := listener.Addr().String()
				assert.NotEmpty(t, addr)
				
				listener.Close()
			}
		})
	}
}

func TestFlow_SendAuthRequest(t *testing.T) {
	tests := []struct {
		name          string
		skipBrowser   bool
		setupServer   func() *httptest.Server
		expectedError string
	}{
		{
			name:        "success browser",
			skipBrowser: false,
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					w.Write([]byte("OK"))
				}))
			},
		},
		{
			name:        "success skip",
			skipBrowser: true,
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify the request is for authorization
					assert.Contains(t, r.URL.String(), "client_id=test-client")
					assert.Contains(t, r.URL.String(), "response_type=code")
					w.WriteHeader(200)
					w.Write([]byte("OK"))
				}))
			},
		},
		{
			name:        "http error",
			skipBrowser: true,
			setupServer: func() *httptest.Server {
				// Return a server that's immediately closed to simulate connection failure
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
				}))
				server.Close() // Close immediately to cause connection error
				return server
			},
			expectedError: "failed to send auth request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			if tt.expectedError == "" {
				defer server.Close()
			}

			provider := &Provider{
				AuthorizationEndpoint: server.URL + "/authorize",
				TokenEndpoint:        server.URL + "/token",
			}

			flow := NewFlow(provider, "test-client", 8080, http.DefaultClient)
			flow.SkipBrowser = true

			err := flow.SendAuthRequest()

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFlow_WaitForCallback_ThreeStep(t *testing.T) {
	tests := []struct {
		name          string
		setupServer   func() *httptest.Server
		sendCallback  func(port int, state string)
		timeout       time.Duration
		expectedError string
	}{
		{
			name: "success",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(200)
					w.Write([]byte(`{"access_token": "test-token", "refresh_token": "test-refresh", "expires_in": 3600, "token_type": "Bearer"}`))
				}))
			},
			sendCallback: func(port int, state string) {
				time.Sleep(50 * time.Millisecond) // Small delay to ensure server is ready
				callbackURL := fmt.Sprintf("http://localhost:%d/callback?code=test-code&state=%s", port, state)
				resp, err := http.Get(callbackURL)
				if err == nil && resp != nil {
					resp.Body.Close()
				}
			},
			timeout: 2 * time.Second,
		},
		{
			name: "timeout",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
				}))
			},
			sendCallback: func(port int, state string) {
				// Don't send callback - will timeout
			},
			timeout:       100 * time.Millisecond,
			expectedError: "authentication timeout",
		},
		{
			name: "bad state",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
				}))
			},
			sendCallback: func(port int, state string) {
				time.Sleep(50 * time.Millisecond)
				callbackURL := fmt.Sprintf("http://localhost:%d/callback?code=test-code&state=wrong-state", port)
				resp, err := http.Get(callbackURL)
				if err == nil && resp != nil {
					resp.Body.Close()
				}
			},
			timeout:       1 * time.Second,
			expectedError: "invalid state parameter",
		},
		{
			name: "context timeout",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
				}))
			},
			sendCallback: func(port int, state string) {
				// Don't send callback
			},
			timeout:       5 * time.Second,
			expectedError: "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			provider := &Provider{
				AuthorizationEndpoint: server.URL + "/authorize",
				TokenEndpoint:        server.URL + "/token",
			}

			// Find a free port
			tempListener, err := net.Listen("tcp", "localhost:0")
			require.NoError(t, err)
			port := tempListener.Addr().(*net.TCPAddr).Port
			tempListener.Close()

			flow := NewFlow(provider, "test-client", port, http.DefaultClient)
			flow.Timeout = tt.timeout

			// Step 1: Start callback server
			listener, err := flow.StartCallbackServer()
			require.NoError(t, err)
			defer listener.Close()

			ctx := context.Background()
			if tt.name == "context timeout" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 50*time.Millisecond)
				defer cancel()
			}

			// Start callback in goroutine
			go tt.sendCallback(port, flow.state)

			// Step 3: Wait for callback
			credentials, err := flow.WaitForCallback(ctx, listener)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, credentials)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, credentials)
				assert.Equal(t, "test-token", credentials.AccessToken)
				assert.Equal(t, "test-refresh", credentials.RefreshToken)
			}
		})
	}
}

func TestFlow_Integration(t *testing.T) {
	// Integration test of the full 3-step flow
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authorize":
			// Mock OAuth provider receiving auth request
			w.WriteHeader(200)
			w.Write([]byte("Authorization in progress"))
			
			// Send callback async (like a real OAuth provider would)
			go func() {
				time.Sleep(100 * time.Millisecond)
				callbackURL := r.URL.Query().Get("redirect_uri")
				state := r.URL.Query().Get("state")
				
				fullURL := callbackURL + "?code=integration-code&state=" + state
				resp, err := http.Get(fullURL)
				if err == nil && resp != nil {
					resp.Body.Close()
				}
			}()
			
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(`{"access_token": "integration-token", "refresh_token": "integration-refresh", "expires_in": 3600, "token_type": "Bearer"}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	provider := &Provider{
		AuthorizationEndpoint: server.URL + "/authorize",
		TokenEndpoint:        server.URL + "/token",
	}

	flow := NewFlow(provider, "test-client", 0, http.DefaultClient)
	flow.SkipBrowser = true

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Step 1: Start callback server
	listener, err := flow.StartCallbackServer()
	require.NoError(t, err)
	defer listener.Close()

	// Update flow port to match actual listener
	actualPort := listener.Addr().(*net.TCPAddr).Port
	flow.redirectPort = actualPort

	// Step 2: Send auth request
	err = flow.SendAuthRequest()
	require.NoError(t, err)

	// Step 3: Wait for callback
	credentials, err := flow.WaitForCallback(ctx, listener)
	require.NoError(t, err)
	assert.NotNil(t, credentials)
	assert.Equal(t, "integration-token", credentials.AccessToken)
	assert.Equal(t, "integration-refresh", credentials.RefreshToken)
}

func TestFlowHandleCallback(t *testing.T) {
	tests := []struct {
		name          string
		requestPath   string
		requestQuery  string
		expectedCode  string
		expectedError string
		simulateError bool
	}{
		{
			name:         "success",
			requestPath:  "/callback",
			requestQuery: "code=auth-code-123&state=test-state",
			expectedCode: "auth-code-123",
		},
		{
			name:          "wrong state",
			requestPath:   "/callback",
			requestQuery:  "code=auth-code&state=wrong-state",
			expectedError: "invalid state parameter",
		},
		{
			name:          "missing code",
			requestPath:   "/callback",
			requestQuery:  "state=test-state",
			expectedError: "no authorization code received",
		},
		{
			name:          "oauth error",
			requestPath:   "/callback",
			requestQuery:  "error=access_denied&error_description=User+denied+access&state=test-state",
			expectedError: "authentication error: access_denied",
		},
		{
			name:          "wrong path",
			requestPath:   "/wrong",
			requestQuery:  "code=auth-code&state=test-state",
			expectedError: "no code received",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flow := &Flow{
				state: "test-state",
			}

			// Create test server that will accept the callback connection
			listener, err := net.Listen("tcp", "localhost:0")
			require.NoError(t, err)
			defer listener.Close()

			codeChan := make(chan string, 1)
			errChan := make(chan error, 1)

			// Start callback handler
			go flow.handleCallback(listener, codeChan, errChan)

			// Give it time to start
			time.Sleep(50 * time.Millisecond)

			// Make callback request
			callbackURL := fmt.Sprintf("http://%s%s?%s", listener.Addr().String(), tt.requestPath, tt.requestQuery)
			resp, err := http.Get(callbackURL)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Check result
			select {
			case code := <-codeChan:
				if tt.expectedError != "" {
					t.Errorf("expected error but got code: %s", code)
				} else {
					assert.Equal(t, tt.expectedCode, code)
				}
			case err := <-errChan:
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				} else {
					t.Errorf("unexpected error: %v", err)
				}
			case <-time.After(time.Second):
				if !tt.simulateError && tt.expectedError == "" {
					t.Error("timeout waiting for callback")
				}
			}
		})
	}
}
