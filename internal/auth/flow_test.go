package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlow_CompleteOIDCAuthFlow(t *testing.T) {
	tests := []struct {
		name string
		// Mocks
		setupHandler func(t *testing.T) http.HandlerFunc

		// Test control
		contextTimeout time.Duration // 0 = use default
		flowTimeout    time.Duration // 0 = use default
		closeServer    bool          // close server before auth request

		// Expected results
		expectAuthErr error                                  // expected error from SendAuthRequest
		expectErr     error                                  // expected error from WaitForCallback
		validateCreds func(t *testing.T, creds *Credentials) // credential validation
	}{
		{
			name: "successful OAuth flow",
			setupHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					t.Logf("OAuth server: %s %s", r.Method, r.URL.Path)
					switch r.URL.Path {
					case "/auth":
						// Authorization endpoint - redirect to callback
						assert.Equal(t, "GET", r.Method)
						callbackURL := r.URL.Query().Get("redirect_uri")
						state := r.URL.Query().Get("state")
						redirectURL := fmt.Sprintf("%s?code=test_code&state=%s", callbackURL, state)
						t.Logf("OAuth server: redirecting to %s", redirectURL)
						http.Redirect(w, r, redirectURL, http.StatusFound)
					case "/token":
						// Token endpoint
						assert.Equal(t, "POST", r.Method)
						assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

						// Parse form data
						err := r.ParseForm()
						require.NoError(t, err)
						assert.Equal(t, "authorization_code", r.FormValue("grant_type"))
						assert.Equal(t, "test_client", r.FormValue("client_id"))
						assert.Equal(t, "test_code", r.FormValue("code"))
						assert.NotEmpty(t, r.FormValue("code_verifier"))

						// Return token response
						response := TokenResponse{
							AccessToken:  "test_access_token",
							TokenType:    "Bearer",
							ExpiresIn:    3600,
							RefreshToken: "test_refresh_token",
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(response)
					default:
						http.NotFound(w, r)
					}
				}
			},
			expectErr: nil,
			validateCreds: func(t *testing.T, creds *Credentials) {
				assert.Equal(t, "test_access_token", creds.AccessToken)
				assert.Equal(t, "test_refresh_token", creds.RefreshToken)
				assert.Equal(t, "Bearer", creds.TokenType)
				assert.False(t, creds.IsExpired())
			},
		},
		{
			name: "OAuth error response",
			setupHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/auth":
						// Authorization endpoint - redirect with error
						callbackURL := r.URL.Query().Get("redirect_uri")
						state := r.URL.Query().Get("state")
						http.Redirect(w, r, fmt.Sprintf("%s?error=access_denied&error_description=User+denied+access&state=%s", callbackURL, state), http.StatusFound)
					default:
						http.NotFound(w, r)
					}
				}
			},
			expectErr: fmt.Errorf("authentication error: access_denied - User denied access"),
			validateCreds: func(t *testing.T, creds *Credentials) {
				assert.Nil(t, creds)
			},
		},
		{
			name: "invalid state parameter",
			setupHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/auth":
						// Authorization endpoint - redirect with wrong state
						callbackURL := r.URL.Query().Get("redirect_uri")
						http.Redirect(w, r, fmt.Sprintf("%s?code=test_code&state=wrong_state", callbackURL), http.StatusFound)
					default:
						http.NotFound(w, r)
					}
				}
			},
			expectErr: fmt.Errorf("invalid state parameter"),
			validateCreds: func(t *testing.T, creds *Credentials) {
				assert.Nil(t, creds)
			},
		},
		{
			name: "missing authorization code",
			setupHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/auth":
						// Authorization endpoint - redirect without code
						callbackURL := r.URL.Query().Get("redirect_uri")
						state := r.URL.Query().Get("state")
						http.Redirect(w, r, fmt.Sprintf("%s?state=%s", callbackURL, state), http.StatusFound)
					default:
						http.NotFound(w, r)
					}
				}
			},
			expectErr: fmt.Errorf("no authorization code received"),
			validateCreds: func(t *testing.T, creds *Credentials) {
				assert.Nil(t, creds)
			},
		},
		{
			name: "token exchange failure",
			setupHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/auth":
						// Authorization endpoint - successful redirect
						callbackURL := r.URL.Query().Get("redirect_uri")
						state := r.URL.Query().Get("state")
						http.Redirect(w, r, fmt.Sprintf("%s?code=test_code&state=%s", callbackURL, state), http.StatusFound)
					case "/token":
						// Token endpoint - return error
						w.WriteHeader(http.StatusBadRequest)
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(map[string]string{
							"error":             "invalid_grant",
							"error_description": "Invalid authorization code",
						})
					default:
						http.NotFound(w, r)
					}
				}
			},
			expectErr: fmt.Errorf("failed to exchange code for tokens: failed to authenticate: invalid_grant - invalid authorization code"),
			validateCreds: func(t *testing.T, creds *Credentials) {
				assert.Nil(t, creds)
			},
		},
		{
			name: "timeout scenario",
			setupHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					// Never respond to auth endpoint to trigger timeout
					if r.URL.Path == "/auth" {
						time.Sleep(200 * time.Millisecond) // Longer than test timeout
					}
					http.NotFound(w, r)
				}
			},
			flowTimeout: 100 * time.Millisecond,
			expectErr:   fmt.Errorf("authentication timeout after 100ms"),
			validateCreds: func(t *testing.T, creds *Credentials) {
				assert.Nil(t, creds)
			},
		},
		{
			name: "context cancellation",
			setupHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					// Never respond to trigger context cancellation
					time.Sleep(200 * time.Millisecond)
					http.NotFound(w, r)
				}
			},
			expectErr: fmt.Errorf("context deadline exceeded"),
			validateCreds: func(t *testing.T, creds *Credentials) {
				assert.Nil(t, creds)
			},
			contextTimeout: 50 * time.Millisecond,
		},
		{
			name: "HTTP request error",
			setupHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					// This handler won't be called because server will be closed
					http.NotFound(w, r)
				}
			},
			closeServer:   true,
			expectAuthErr: fmt.Errorf("failed to send auth request"),
			validateCreds: func(t *testing.T, creds *Credentials) {
				assert.Nil(t, creds)
			},
		},
		{
			name: "token without expiration",
			setupHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/auth":
						callbackURL := r.URL.Query().Get("redirect_uri")
						state := r.URL.Query().Get("state")
						http.Redirect(w, r, fmt.Sprintf("%s?code=test_code&state=%s", callbackURL, state), http.StatusFound)
					case "/token":
						// Token without expires_in - should be rejected
						response := TokenResponse{
							AccessToken:  "test_access_token",
							TokenType:    "Bearer",
							RefreshToken: "test_refresh_token",
							// ExpiresIn: 0 (no expiration)
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(response)
					default:
						http.NotFound(w, r)
					}
				}
			},
			expectErr: fmt.Errorf("token without expiration is not allowed for security reasons"),
			validateCreds: func(t *testing.T, creds *Credentials) {
				assert.Nil(t, creds)
			},
		},
		{
			name: "token response with missing token_type",
			setupHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/auth":
						callbackURL := r.URL.Query().Get("redirect_uri")
						state := r.URL.Query().Get("state")
						http.Redirect(w, r, fmt.Sprintf("%s?code=test_code&state=%s", callbackURL, state), http.StatusFound)
					case "/token":
						// Token without token_type - should default to Bearer
						response := TokenResponse{
							AccessToken:  "test_access_token",
							ExpiresIn:    3600,
							RefreshToken: "test_refresh_token",
							// TokenType is empty (missing)
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(response)
					default:
						http.NotFound(w, r)
					}
				}
			},
			validateCreds: func(t *testing.T, creds *Credentials) {
				assert.Equal(t, "test_access_token", creds.AccessToken)
				assert.Equal(t, "Bearer", creds.TokenType) // Should default to Bearer
				assert.False(t, creds.IsExpired())
			},
		},
	}

	// Test malformed auth URL parsing
	t.Run("malformed auth URL", func(t *testing.T) {
		// Create provider that returns malformed URL
		provider := &OIDCProvider{
			AuthorizationEndpoint: "ht!tp://bad-url-scheme",
			TokenEndpoint:         "/token",
		}

		httpClient := &http.Client{Timeout: 1 * time.Second}
		flow := NewFlow("test_client", 0, httpClient, WithProvider(provider))
		flow.SkipBrowser = true

		// Start callback server
		err := flow.StartCallbackServer()
		require.NoError(t, err)

		// SendAuthRequest should fail with URL parse error
		err = flow.SendAuthRequest()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse auth URL")
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test server with handler
			handler := tt.setupHandler(t)
			server := httptest.NewServer(handler)
			defer server.Close()
			t.Logf("OAuth server: %s", server.URL)

			// Close server early if requested
			if tt.closeServer {
				server.Close()
			}

			// Create real OIDC provider
			provider := &OIDCProvider{
				AuthorizationEndpoint: server.URL + "/auth",
				TokenEndpoint:         server.URL + "/token",
			}

			// Create real HTTP client
			httpClient := &http.Client{Timeout: 1 * time.Second}

			// Create flow with timeout
			flowTimeout := 100 * time.Millisecond
			if tt.flowTimeout > 0 {
				flowTimeout = tt.flowTimeout
			}

			flow := NewFlow("test_client", 0, httpClient, WithProvider(provider), WithStateGenerator(func() string {
				return "test_state"
			}))
			flow.timeout = flowTimeout
			flow.SkipBrowser = true

			// Start callback server
			err := flow.StartCallbackServer()
			require.NoError(t, err)
			t.Logf("Callback server: localhost:%d", flow.callbackPort)

			// Build and log auth URL
			authURL := flow.GetAuthURL()
			t.Logf("Auth URL: %s", authURL)

			// Send auth request
			err = flow.SendAuthRequest()
			if tt.expectAuthErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectAuthErr.Error())
				t.Logf("Auth request: error - %v", err)
			} else {
				require.NoError(t, err)
				t.Logf("Auth request: sent")
			}

			// Wait for OAuth callback (only if auth request succeeded)
			var creds *Credentials
			if tt.expectAuthErr == nil {
				contextTimeout := flowTimeout // Use flow timeout by default
				if tt.contextTimeout > 0 {
					contextTimeout = tt.contextTimeout
				}
				ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
				defer cancel()

				creds, err = flow.WaitForCallback(ctx)
				if err != nil {
					t.Logf("OAuth result: error - %v", err)
				} else {
					t.Logf("OAuth result: success - got credentials")
				}

				// Validate results
				if tt.expectErr != nil {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectErr.Error())
				} else {
					assert.NoError(t, err)
				}
			}

			tt.validateCreds(t, creds)
		})
	}
}
