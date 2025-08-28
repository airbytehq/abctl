package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredentials_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired - future time",
			expiresAt: time.Now().Add(2 * time.Hour),
			want:      false,
		},
		{
			name:      "expired - past time",
			expiresAt: time.Now().Add(-2 * time.Hour),
			want:      true,
		},
		{
			name:      "not expired - within buffer window",
			expiresAt: time.Now().Add(30 * time.Second),
			want:      true, // Should be true because of 1-minute buffer
		},
		{
			name:      "not expired - just outside buffer window",
			expiresAt: time.Now().Add(90 * time.Second),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Credentials{
				ExpiresAt: tt.expiresAt,
			}
			assert.Equal(t, tt.want, c.IsExpired())
		})
	}
}

func TestCredentials_JSON(t *testing.T) {
	t.Run("ToJSON and FromJSON round trip", func(t *testing.T) {
		original := &Credentials{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			TokenType:    "Bearer",
			ExpiresAt:    time.Now().Add(time.Hour).Truncate(time.Second),
		}

		// Serialize to JSON
		data, err := original.ToJSON()
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		// Deserialize from JSON
		restored, err := CredentialsFromJSON(data)
		require.NoError(t, err)

		// Compare - need to handle time comparison specially
		assert.Equal(t, original.AccessToken, restored.AccessToken)
		assert.Equal(t, original.RefreshToken, restored.RefreshToken)
		assert.Equal(t, original.TokenType, restored.TokenType)
		assert.WithinDuration(t, original.ExpiresAt, restored.ExpiresAt, time.Second)
	})

	t.Run("FromJSON with invalid JSON", func(t *testing.T) {
		invalidJSON := []byte("not-valid-json")
		creds, err := CredentialsFromJSON(invalidJSON)
		assert.Error(t, err)
		assert.Nil(t, creds)
	})

	t.Run("FromJSON with empty data", func(t *testing.T) {
		emptyJSON := []byte("{}")
		creds, err := CredentialsFromJSON(emptyJSON)
		require.NoError(t, err)
		assert.NotNil(t, creds)
		assert.Empty(t, creds.AccessToken)
		assert.Empty(t, creds.RefreshToken)
	})
}

func TestDiscoverProvider(t *testing.T) {
	tests := []struct {
		name         string
		responseCode int
		responseBody interface{}
		wantErr      bool
		errContains  string
		validate     func(*testing.T, *Provider)
	}{
		{
			name:         "successful discovery",
			responseCode: http.StatusOK,
			responseBody: map[string]string{
				"issuer":                 "https://auth.example.com",
				"authorization_endpoint": "https://auth.example.com/authorize",
				"token_endpoint":         "https://auth.example.com/token",
				"userinfo_endpoint":      "https://auth.example.com/userinfo",
				"jwks_uri":               "https://auth.example.com/jwks",
			},
			wantErr: false,
			validate: func(t *testing.T, p *Provider) {
				assert.Equal(t, "https://auth.example.com", p.Issuer)
				assert.Equal(t, "https://auth.example.com/authorize", p.AuthorizationEndpoint)
				assert.Equal(t, "https://auth.example.com/token", p.TokenEndpoint)
				assert.Equal(t, "https://auth.example.com/userinfo", p.UserinfoEndpoint)
				assert.Equal(t, "https://auth.example.com/jwks", p.JwksURI)
			},
		},
		{
			name:         "discovery returns 404",
			responseCode: http.StatusNotFound,
			responseBody: "Not Found",
			wantErr:      true,
			errContains:  "discovery failed with status 404",
		},
		{
			name:         "discovery returns invalid JSON",
			responseCode: http.StatusOK,
			responseBody: "not-json",
			wantErr:      true,
			errContains:  "failed to decode provider configuration",
		},
		{
			name:         "discovery returns partial config",
			responseCode: http.StatusOK,
			responseBody: map[string]string{
				"issuer":                 "https://auth.example.com",
				"authorization_endpoint": "https://auth.example.com/authorize",
				"token_endpoint":         "https://auth.example.com/token",
			},
			wantErr: false,
			validate: func(t *testing.T, p *Provider) {
				assert.Equal(t, "https://auth.example.com", p.Issuer)
				assert.Equal(t, "https://auth.example.com/authorize", p.AuthorizationEndpoint)
				assert.Equal(t, "https://auth.example.com/token", p.TokenEndpoint)
				assert.Empty(t, p.UserinfoEndpoint)
				assert.Empty(t, p.JwksURI)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/.well-known/openid-configuration", r.URL.Path)
				assert.Equal(t, "GET", r.Method)

				w.WriteHeader(tt.responseCode)
				if tt.responseBody != nil {
					if str, ok := tt.responseBody.(string); ok {
						w.Write([]byte(str))
					} else {
						json.NewEncoder(w).Encode(tt.responseBody)
					}
				}
			}))
			defer server.Close()

			provider, err := DiscoverProvider(context.Background(), server.URL)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, provider)
				if tt.validate != nil {
					tt.validate(t, provider)
				}
			}
		})
	}
}

func TestExchangeCodeForTokens(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		redirectURI  string
		codeVerifier string
		responseCode int
		responseBody interface{}
		wantErr      bool
		errContains  string
		validate     func(*testing.T, *TokenResponse)
	}{
		{
			name:         "successful token exchange",
			code:         "test-code",
			redirectURI:  "http://localhost:51004/callback",
			codeVerifier: "test-verifier",
			responseCode: http.StatusOK,
			responseBody: map[string]interface{}{
				"access_token":  "test-access-token",
				"refresh_token": "test-refresh-token",
				"token_type":    "Bearer",
				"expires_in":    3600,
				"id_token":      "test-id-token",
			},
			wantErr: false,
			validate: func(t *testing.T, tr *TokenResponse) {
				assert.Equal(t, "test-access-token", tr.AccessToken)
				assert.Equal(t, "test-refresh-token", tr.RefreshToken)
				assert.Equal(t, "Bearer", tr.TokenType)
				assert.Equal(t, 3600, tr.ExpiresIn)
				assert.Equal(t, "test-id-token", tr.IDToken)
			},
		},
		{
			name:         "token exchange without refresh token",
			code:         "test-code",
			redirectURI:  "http://localhost:51004/callback",
			codeVerifier: "",
			responseCode: http.StatusOK,
			responseBody: map[string]interface{}{
				"access_token": "test-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			},
			wantErr: false,
			validate: func(t *testing.T, tr *TokenResponse) {
				assert.Equal(t, "test-access-token", tr.AccessToken)
				assert.Empty(t, tr.RefreshToken)
				assert.Equal(t, "Bearer", tr.TokenType)
				assert.Equal(t, 3600, tr.ExpiresIn)
			},
		},
		{
			name:         "token exchange returns error",
			code:         "invalid-code",
			redirectURI:  "http://localhost:51004/callback",
			responseCode: http.StatusBadRequest,
			responseBody: map[string]string{
				"error":             "invalid_grant",
				"error_description": "Invalid authorization code",
			},
			wantErr:     true,
			errContains: "invalid_grant - Invalid authorization code",
		},
		{
			name:         "token exchange returns invalid JSON",
			code:         "test-code",
			redirectURI:  "http://localhost:51004/callback",
			responseCode: http.StatusOK,
			responseBody: "not-json",
			wantErr:      true,
			errContains:  "failed to decode token response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/token", r.URL.Path)
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

				// Parse form data
				err := r.ParseForm()
				require.NoError(t, err)
				assert.Equal(t, "authorization_code", r.FormValue("grant_type"))
				assert.Equal(t, "test-client", r.FormValue("client_id"))
				assert.Equal(t, tt.code, r.FormValue("code"))
				assert.Equal(t, tt.redirectURI, r.FormValue("redirect_uri"))
				if tt.codeVerifier != "" {
					assert.Equal(t, tt.codeVerifier, r.FormValue("code_verifier"))
				}

				w.WriteHeader(tt.responseCode)
				if tt.responseBody != nil {
					if str, ok := tt.responseBody.(string); ok {
						w.Write([]byte(str))
					} else {
						json.NewEncoder(w).Encode(tt.responseBody)
					}
				}
			}))
			defer server.Close()

			provider := &Provider{
				TokenEndpoint: server.URL + "/token",
			}

			tokens, err := ExchangeCodeForTokens(
				context.Background(),
				provider,
				"test-client",
				tt.code,
				tt.redirectURI,
				tt.codeVerifier,
			)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, tokens)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, tokens)
				if tt.validate != nil {
					tt.validate(t, tokens)
				}
			}
		})
	}
}

func TestRefreshAccessToken(t *testing.T) {
	tests := []struct {
		name         string
		refreshToken string
		responseCode int
		responseBody interface{}
		wantErr      bool
		errContains  string
		validate     func(*testing.T, *TokenResponse)
	}{
		{
			name:         "successful refresh",
			refreshToken: "test-refresh-token",
			responseCode: http.StatusOK,
			responseBody: map[string]interface{}{
				"access_token":  "new-access-token",
				"refresh_token": "new-refresh-token",
				"token_type":    "Bearer",
				"expires_in":    3600,
			},
			wantErr: false,
			validate: func(t *testing.T, tr *TokenResponse) {
				assert.Equal(t, "new-access-token", tr.AccessToken)
				assert.Equal(t, "new-refresh-token", tr.RefreshToken)
				assert.Equal(t, "Bearer", tr.TokenType)
				assert.Equal(t, 3600, tr.ExpiresIn)
			},
		},
		{
			name:         "refresh without new refresh token",
			refreshToken: "test-refresh-token",
			responseCode: http.StatusOK,
			responseBody: map[string]interface{}{
				"access_token": "new-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			},
			wantErr: false,
			validate: func(t *testing.T, tr *TokenResponse) {
				assert.Equal(t, "new-access-token", tr.AccessToken)
				assert.Empty(t, tr.RefreshToken)
				assert.Equal(t, "Bearer", tr.TokenType)
				assert.Equal(t, 3600, tr.ExpiresIn)
			},
		},
		{
			name:         "refresh returns error",
			refreshToken: "expired-refresh-token",
			responseCode: http.StatusBadRequest,
			responseBody: map[string]string{
				"error":             "invalid_grant",
				"error_description": "Refresh token expired",
			},
			wantErr:     true,
			errContains: "invalid_grant - Refresh token expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/token", r.URL.Path)
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

				// Parse form data
				err := r.ParseForm()
				require.NoError(t, err)
				assert.Equal(t, "refresh_token", r.FormValue("grant_type"))
				assert.Equal(t, "test-client", r.FormValue("client_id"))
				assert.Equal(t, tt.refreshToken, r.FormValue("refresh_token"))

				w.WriteHeader(tt.responseCode)
				if tt.responseBody != nil {
					json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			provider := &Provider{
				TokenEndpoint: server.URL + "/token",
			}

			tokens, err := RefreshAccessToken(
				context.Background(),
				provider,
				"test-client",
				tt.refreshToken,
			)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, tokens)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, tokens)
				if tt.validate != nil {
					tt.validate(t, tokens)
				}
			}
		})
	}
}

func TestDoTokenRequest(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		// Server that never responds
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		data := url.Values{}
		data.Set("grant_type", "authorization_code")

		tokens, err := doTokenRequest(ctx, server.URL, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
		assert.Nil(t, tokens)
	})

	t.Run("server returns 500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error":             "server_error",
				"error_description": "Internal server error",
			})
		}))
		defer server.Close()

		data := url.Values{}
		data.Set("grant_type", "authorization_code")

		tokens, err := doTokenRequest(context.Background(), server.URL, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server_error")
		assert.Nil(t, tokens)
	})
}
