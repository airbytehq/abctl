package auth

import (
	"context"
	"fmt"
	"io"
	stdhttp "net/http"
	"strings"
	"testing"
	"time"

	"github.com/airbytehq/abctl/internal/http"
	httpmock "github.com/airbytehq/abctl/internal/http/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name           string
		updateHook     CredentialsUpdateHook
		expectNilHook  bool
	}{
		{
			name:       "success",
			updateHook: func(*Credentials) error { return nil },
		},
		{
			name:          "nil hook",
			updateHook:    nil,
			expectNilHook: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
			provider := &Provider{TokenEndpoint: "https://auth.example.com/token"}
			credentials := &Credentials{AccessToken: "token123"}

			client := NewClient(provider, "client-id", credentials, mockHTTP, tt.updateHook)

			assert.NotNil(t, client)
			assert.Equal(t, provider, client.provider)
			assert.Equal(t, "client-id", client.clientID)
			assert.Equal(t, credentials, client.credentials)
			assert.Equal(t, mockHTTP, client.httpClient)

			if tt.expectNilHook {
				// Test that default hook returns ErrNoUpdateHook
				err := client.updateHook(&Credentials{})
				assert.Equal(t, ErrNoUpdateHook, err)
			}
		})
	}
}

func TestClientDo(t *testing.T) {
	tests := []struct {
		name           string
		credentials    *Credentials
		setupMocks     func(ctrl *gomock.Controller) (http.HTTPDoer, CredentialsUpdateHook)
		expectedError  string
	}{
		{
			name: "success",
			credentials: &Credentials{
				AccessToken: "valid-token",
				TokenType:   "Bearer",
				ExpiresAt:   time.Now().Add(time.Hour),
			},
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, CredentialsUpdateHook) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("success")),
					Header:     make(stdhttp.Header),
				}, nil).Times(1)

				updateHook := func(*Credentials) error { return nil }
				return mockHTTP, updateHook
			},
		},
		{
			name: "expired token refresh",
			credentials: &Credentials{
				AccessToken:  "expired-token",
				RefreshToken: "refresh123",
				TokenType:    "Bearer",
				ExpiresAt:    time.Now().Add(-time.Hour), // Expired
			},
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, CredentialsUpdateHook) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				
				// First call for refresh token request
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(`{
						"access_token": "new-token",
						"refresh_token": "new-refresh",
						"token_type": "Bearer",
						"expires_in": 3600
					}`)),
					Header: make(stdhttp.Header),
				}, nil).Times(1)

				// Second call for actual request
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("success")),
					Header:     make(stdhttp.Header),
				}, nil).Times(1)

				updateHook := func(*Credentials) error { return nil }
				return mockHTTP, updateHook
			},
		},
		{
			name: "401 retry",
			credentials: &Credentials{
				AccessToken:  "token-becomes-invalid",
				RefreshToken: "refresh123",
				TokenType:    "Bearer",
				ExpiresAt:    time.Now().Add(time.Hour), // Not expired initially
			},
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, CredentialsUpdateHook) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				
				// First call returns 401
				resp401 := &stdhttp.Response{
					StatusCode: 401,
					Body:       io.NopCloser(strings.NewReader("unauthorized")),
					Header:     make(stdhttp.Header),
				}
				mockHTTP.EXPECT().Do(gomock.Any()).Return(resp401, nil).Times(1)

				// Refresh token request
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(`{
						"access_token": "refreshed-token",
						"token_type": "Bearer",
						"expires_in": 3600
					}`)),
					Header: make(stdhttp.Header),
				}, nil).Times(1)

				// Retry request succeeds
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("success after refresh")),
					Header:     make(stdhttp.Header),
				}, nil).Times(1)

				updateHook := func(*Credentials) error { return nil }
				return mockHTTP, updateHook
			},
		},
		{
			name: "401 empty token type",
			credentials: &Credentials{
				AccessToken:  "token-becomes-invalid",
				RefreshToken: "refresh123",
				TokenType:    "Bearer",
				ExpiresAt:    time.Now().Add(time.Hour),
			},
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, CredentialsUpdateHook) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				
				// First call returns 401
				resp401 := &stdhttp.Response{
					StatusCode: 401,
					Body:       io.NopCloser(strings.NewReader("unauthorized")),
					Header:     make(stdhttp.Header),
				}
				mockHTTP.EXPECT().Do(gomock.Any()).Return(resp401, nil).Times(1)

				// Refresh returns empty token_type
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(`{
						"access_token": "refreshed-token",
						"expires_in": 3600
					}`)),
					Header: make(stdhttp.Header),
				}, nil).Times(1)

				// Retry request succeeds
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("success")),
					Header:     make(stdhttp.Header),
				}, nil).Times(1)

				updateHook := func(creds *Credentials) error {
					// Verify default was set
					assert.Equal(t, "Bearer", creds.TokenType)
					return nil
				}
				return mockHTTP, updateHook
			},
		},
		{
			name: "no refresh token",
			credentials: &Credentials{
				AccessToken: "expired-token",
				TokenType:   "Bearer",
				ExpiresAt:   time.Now().Add(-time.Hour),
				// No RefreshToken
			},
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, CredentialsUpdateHook) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				updateHook := func(*Credentials) error { return nil }
				return mockHTTP, updateHook
			},
			expectedError: "not authenticated - please run 'airbox auth login' first",
		},
		{
			name: "refresh error",
			credentials: &Credentials{
				AccessToken:  "expired-token",
				RefreshToken: "invalid-refresh",
				TokenType:    "Bearer",
				ExpiresAt:    time.Now().Add(-time.Hour),
			},
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, CredentialsUpdateHook) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				
				// Refresh token request fails
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 400,
					Body:       io.NopCloser(strings.NewReader(`{"error": "invalid_grant"}`)),
					Header:     make(stdhttp.Header),
				}, nil).Times(1)

				updateHook := func(*Credentials) error { return nil }
				return mockHTTP, updateHook
			},
			expectedError: "token refresh failed",
		},
		{
			name: "hook error",
			credentials: &Credentials{
				AccessToken:  "expired-token",
				RefreshToken: "refresh123",
				TokenType:    "Bearer",
				ExpiresAt:    time.Now().Add(-time.Hour),
			},
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, CredentialsUpdateHook) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				
				// Successful refresh
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(`{
						"access_token": "new-token",
						"token_type": "Bearer",
						"expires_in": 3600
					}`)),
					Header: make(stdhttp.Header),
				}, nil).Times(1)

				updateHook := func(*Credentials) error {
					return fmt.Errorf("failed to save credentials")
				}
				return mockHTTP, updateHook
			},
			expectedError: "credentials update hook failed",
		},
		{
			name: "network error",
			credentials: &Credentials{
				AccessToken: "valid-token",
				TokenType:   "Bearer",
				ExpiresAt:   time.Now().Add(time.Hour),
			},
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, CredentialsUpdateHook) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockHTTP.EXPECT().Do(gomock.Any()).Return(nil, fmt.Errorf("network error")).Times(1)

				updateHook := func(*Credentials) error { return nil }
				return mockHTTP, updateHook
			},
			expectedError: "network error",
		},
		{
			name: "401 refresh fails",
			credentials: &Credentials{
				AccessToken:  "invalid-token",
				RefreshToken: "bad-refresh",
				TokenType:    "Bearer",
				ExpiresAt:    time.Now().Add(time.Hour),
			},
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, CredentialsUpdateHook) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				
				// First call returns 401
				resp401 := &stdhttp.Response{
					StatusCode: 401,
					Body:       io.NopCloser(strings.NewReader("unauthorized")),
					Header:     make(stdhttp.Header),
				}
				mockHTTP.EXPECT().Do(gomock.Any()).Return(resp401, nil).Times(1)

				// Refresh attempt fails
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 400,
					Body:       io.NopCloser(strings.NewReader(`{"error": "invalid_grant"}`)),
					Header:     make(stdhttp.Header),
				}, nil).Times(1)

				updateHook := func(*Credentials) error { return nil }
				return mockHTTP, updateHook
			},
			expectedError: "failed to refresh token after 401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHTTP, updateHook := tt.setupMocks(ctrl)
			provider := &Provider{
				TokenEndpoint: "https://auth.example.com/token",
			}

			client := NewClient(provider, "client-id", tt.credentials, mockHTTP, updateHook)

			req, err := stdhttp.NewRequest("GET", "https://api.example.com/test", nil)
			require.NoError(t, err)

			resp, err := client.Do(req)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, resp)
				if resp != nil {
					resp.Body.Close()
				}

				// Verify Authorization header was set
				authHeader := req.Header.Get("Authorization")
				assert.Contains(t, authHeader, "Bearer")
			}
		})
	}
}

func TestClientGetCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
	provider := &Provider{TokenEndpoint: "https://auth.example.com/token"}
	
	originalCreds := &Credentials{
		AccessToken:  "original-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	updateHook := func(*Credentials) error { return nil }
	client := NewClient(provider, "client-id", originalCreds, mockHTTP, updateHook)

	// Get credentials copy
	credsCopy := client.GetCredentials()

	// Verify it's a copy, not the same reference
	assert.NotSame(t, originalCreds, credsCopy)
	
	// Verify values are the same
	assert.Equal(t, originalCreds.AccessToken, credsCopy.AccessToken)
	assert.Equal(t, originalCreds.RefreshToken, credsCopy.RefreshToken)
	assert.Equal(t, originalCreds.TokenType, credsCopy.TokenType)
	assert.Equal(t, originalCreds.ExpiresAt, credsCopy.ExpiresAt)

	// Modify copy and verify original is unchanged
	credsCopy.AccessToken = "modified-token"
	assert.NotEqual(t, originalCreds.AccessToken, credsCopy.AccessToken)
	assert.Equal(t, "original-token", originalCreds.AccessToken)
}

func TestClientEnsureValidToken(t *testing.T) {
	tests := []struct {
		name          string
		credentials   *Credentials
		setupMocks    func(ctrl *gomock.Controller) (http.HTTPDoer, CredentialsUpdateHook)
		expectedError string
	}{
		{
			name: "valid token",
			credentials: &Credentials{
				AccessToken: "valid-token",
				ExpiresAt:   time.Now().Add(time.Hour),
			},
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, CredentialsUpdateHook) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				updateHook := func(*Credentials) error { return nil }
				return mockHTTP, updateHook
			},
		},
		{
			name: "no refresh token",
			credentials: &Credentials{
				AccessToken: "expired-token",
				ExpiresAt:   time.Now().Add(-time.Hour),
				// No RefreshToken
			},
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, CredentialsUpdateHook) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				updateHook := func(*Credentials) error { return nil }
				return mockHTTP, updateHook
			},
			expectedError: "not authenticated - please run 'airbox auth login' first",
		},
		{
			name: "refresh success",
			credentials: &Credentials{
				AccessToken:  "expired-token",
				RefreshToken: "refresh123",
				ExpiresAt:    time.Now().Add(-time.Hour),
			},
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, CredentialsUpdateHook) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(`{
						"access_token": "new-token",
						"refresh_token": "new-refresh",
						"token_type": "Bearer",
						"expires_in": 3600
					}`)),
					Header: make(stdhttp.Header),
				}, nil).Times(1)

				updateHook := func(*Credentials) error { return nil }
				return mockHTTP, updateHook
			},
		},
		{
			name: "refresh empty token type",
			credentials: &Credentials{
				AccessToken:  "expired-token",
				RefreshToken: "refresh123",
				ExpiresAt:    time.Now().Add(-time.Hour),
			},
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, CredentialsUpdateHook) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				
				// Return empty token_type to trigger default
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(`{
						"access_token": "new-token",
						"refresh_token": "new-refresh",
						"expires_in": 3600
					}`)),
					Header: make(stdhttp.Header),
				}, nil).Times(1)

				updateHook := func(creds *Credentials) error {
					// Verify TokenType was set to default
					assert.Equal(t, "Bearer", creds.TokenType)
					return nil
				}
				return mockHTTP, updateHook
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHTTP, updateHook := tt.setupMocks(ctrl)
			provider := &Provider{TokenEndpoint: "https://auth.example.com/token"}

			client := NewClient(provider, "client-id", tt.credentials, mockHTTP, updateHook)

			err := client.ensureValidToken(context.Background())

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}