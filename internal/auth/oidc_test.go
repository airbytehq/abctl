package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	httpmock "github.com/airbytehq/abctl/internal/http/mock"
)

func TestOIDCProvider_RefreshToken(t *testing.T) {
	tests := []struct {
		name         string
		refreshToken string
		setupHTTP    func(mock *httpmock.MockHTTPDoer)
		expectErr    error
	}{
		{
			name:         "success",
			refreshToken: "valid-refresh-token",
			setupHTTP: func(mock *httpmock.MockHTTPDoer) {
				response := `{
					"access_token": "refreshed-oidc-token",
					"refresh_token": "new-oidc-refresh-token",
					"token_type": "Bearer",
					"expires_in": 3600
				}`
				mock.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(response)),
				}, nil)
			},
		},
		{
			name:         "empty refresh token returns error",
			refreshToken: "",
			setupHTTP: func(mock *httpmock.MockHTTPDoer) {
				response := `{
					"error": "invalid_request",
					"error_description": "refresh_token is required"
				}`
				mock.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 400,
					Body:       io.NopCloser(strings.NewReader(response)),
				}, nil)
			},
			expectErr: errors.New("failed to authenticate: invalid_request - refresh_token is required"),
		},
		{
			name:         "invalid refresh token",
			refreshToken: "invalid-refresh-token",
			setupHTTP: func(mock *httpmock.MockHTTPDoer) {
				response := `{
					"error": "invalid_grant",
					"error_description": "The provided authorization grant is invalid"
				}`
				mock.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 400,
					Body:       io.NopCloser(strings.NewReader(response)),
				}, nil)
			},
			expectErr: errors.New("failed to authenticate: invalid_grant - the provided authorization grant is invalid"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHTTPClient := httpmock.NewMockHTTPDoer(ctrl)
			tt.setupHTTP(mockHTTPClient)

			provider := &OIDCProvider{
				TokenEndpoint: "https://example.com/token",
			}

			tokenResp, err := provider.RefreshToken(context.Background(), tt.refreshToken, mockHTTPClient)

			if tt.expectErr != nil {
				assert.Equal(t, tt.expectErr, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tokenResp)
			}
		})
	}
}

func TestOIDCProvider_GetTokenEndpoint(t *testing.T) {
	provider := &OIDCProvider{
		TokenEndpoint: "https://example.com/token",
	}
	endpoint := provider.GetTokenEndpoint()

	assert.Equal(t, "https://example.com/token", endpoint)
}

func TestOIDCProvider_AuthEndpointHandler(t *testing.T) {
	provider := &OIDCProvider{
		AuthorizationEndpoint: "https://example.com/auth",
	}
	handler := provider.AuthEndpointHandler()

	require.NotNil(t, handler)

	url := handler("test-client-id", "https://example.com/callback", "test-state", "test-challenge")

	expected := "https://example.com/auth?client_id=test-client-id&code_challenge=test-challenge&code_challenge_method=S256&redirect_uri=https%3A%2F%2Fexample.com%2Fcallback&response_type=code&scope=openid+profile+email+offline_access&state=test-state"
	assert.Equal(t, expected, url)
}

func TestNewOIDCProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTPClient := httpmock.NewMockHTTPDoer(ctrl)
	mockStore := NewMockCredentialsStore(ctrl)

	provider := NewOIDCProvider(mockHTTPClient, mockStore)

	assert.NotNil(t, provider)
	assert.Equal(t, mockHTTPClient, provider.httpClient)
	assert.Equal(t, mockStore, provider.store)
}

func TestOIDCProvider_Load(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockCredentialsStore(ctrl)
	provider := &OIDCProvider{store: mockStore}

	expectedCreds := &Credentials{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}
	mockStore.EXPECT().Load().Return(expectedCreds, nil)

	creds, err := provider.Load()

	assert.NoError(t, err)
	assert.Equal(t, expectedCreds, creds)
}

func TestOIDCProvider_Save(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockCredentialsStore(ctrl)
	provider := &OIDCProvider{store: mockStore}

	creds := &Credentials{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}
	mockStore.EXPECT().Save(creds).Return(nil)

	err := provider.Save(creds)

	assert.NoError(t, err)
}

func TestOIDCProvider_SetCredentialsStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockCredentialsStore(ctrl)
	provider := &OIDCProvider{}

	provider.SetCredentialsStore(mockStore)

	assert.Equal(t, mockStore, provider.store)
}

func TestOIDCProvider_Do(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(ctrl *gomock.Controller) (*httpmock.MockHTTPDoer, *MockCredentialsStore)
		expectErr  error
	}{
		{
			name: "success with valid token",
			setupMocks: func(ctrl *gomock.Controller) (*httpmock.MockHTTPDoer, *MockCredentialsStore) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockStore := NewMockCredentialsStore(ctrl)

				mockStore.EXPECT().Load().Return(&Credentials{
					AccessToken: "valid-token",
					TokenType:   "Bearer",
					ExpiresAt:   time.Now().Add(time.Hour),
				}, nil)

				mockHTTP.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("success")),
				}, nil)

				return mockHTTP, mockStore
			},
		},
		{
			name: "retry on 401 with token refresh",
			setupMocks: func(ctrl *gomock.Controller) (*httpmock.MockHTTPDoer, *MockCredentialsStore) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockStore := NewMockCredentialsStore(ctrl)

				// First call - return valid token with refresh token
				mockStore.EXPECT().Load().Return(&Credentials{
					AccessToken:  "expired-token",
					RefreshToken: "refresh-token",
					TokenType:    "Bearer",
					ExpiresAt:    time.Now().Add(time.Hour),
				}, nil)

				// First HTTP request returns 401
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 401,
					Body:       io.NopCloser(strings.NewReader("unauthorized")),
				}, nil)

				// Mark token as expired and refresh
				mockStore.EXPECT().Save(gomock.Any()).Return(nil)
				mockStore.EXPECT().Load().Return(&Credentials{
					AccessToken:  "expired-token",
					RefreshToken: "refresh-token",
					TokenType:    "Bearer",
					ExpiresAt:    time.Now().Add(-time.Hour),
				}, nil)

				// Refresh token call
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"access_token": "fresh-token", "refresh_token": "new-refresh-token", "expires_in": 3600}`)),
				}, nil)
				mockStore.EXPECT().Save(gomock.Any()).Return(nil)

				// Second HTTP request with fresh token succeeds
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("success")),
				}, nil)

				return mockHTTP, mockStore
			},
		},
		{
			name: "error getting valid token",
			setupMocks: func(ctrl *gomock.Controller) (*httpmock.MockHTTPDoer, *MockCredentialsStore) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockStore := NewMockCredentialsStore(ctrl)

				mockStore.EXPECT().Load().Return(nil, errors.New("store error"))

				return mockHTTP, mockStore
			},
			expectErr: errors.New("failed to get valid token: no valid credentials available"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHTTP, mockStore := tt.setupMocks(ctrl)

			provider := &OIDCProvider{
				TokenEndpoint: "https://example.com/token",
				ClientID:      "test-client-id",
				httpClient:    mockHTTP,
				store:         mockStore,
			}

			req, _ := http.NewRequest("GET", "https://example.com/api", nil)
			resp, err := provider.Do(req)

			if tt.expectErr == nil {
				require.NoError(t, err)
				assert.NotNil(t, resp)
				if resp != nil {
					_ = resp.Body.Close()
				}
				return
			}
			assert.Equal(t, tt.expectErr.Error(), err.Error())
		})
	}
}

func TestOIDCProvider_ensureValidToken(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(ctrl *gomock.Controller) (*httpmock.MockHTTPDoer, *MockCredentialsStore)
		expectErr  error
	}{
		{
			name: "valid token exists",
			setupMocks: func(ctrl *gomock.Controller) (*httpmock.MockHTTPDoer, *MockCredentialsStore) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockStore := NewMockCredentialsStore(ctrl)

				mockStore.EXPECT().Load().Return(&Credentials{
					AccessToken: "valid-token",
					TokenType:   "Bearer",
					ExpiresAt:   time.Now().Add(time.Hour),
				}, nil)

				return mockHTTP, mockStore
			},
		},
		{
			name: "expired token with refresh",
			setupMocks: func(ctrl *gomock.Controller) (*httpmock.MockHTTPDoer, *MockCredentialsStore) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockStore := NewMockCredentialsStore(ctrl)

				mockStore.EXPECT().Load().Return(&Credentials{
					AccessToken:  "expired-token",
					RefreshToken: "refresh-token",
					TokenType:    "Bearer",
					ExpiresAt:    time.Now().Add(-time.Hour),
				}, nil)

				mockHTTP.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"access_token": "fresh-token", "refresh_token": "new-refresh-token", "expires_in": 3600, "token_type": "Bearer"}`)),
				}, nil)
				mockStore.EXPECT().Save(gomock.Any()).Return(nil)

				return mockHTTP, mockStore
			},
		},
		{
			name: "expired token with refresh failure",
			setupMocks: func(ctrl *gomock.Controller) (*httpmock.MockHTTPDoer, *MockCredentialsStore) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockStore := NewMockCredentialsStore(ctrl)

				mockStore.EXPECT().Load().Return(&Credentials{
					AccessToken:  "expired-token",
					RefreshToken: "invalid-refresh-token",
					TokenType:    "Bearer",
					ExpiresAt:    time.Now().Add(-time.Hour),
				}, nil)

				mockHTTP.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 400,
					Body:       io.NopCloser(strings.NewReader(`{"error": "invalid_grant", "error_description": "The provided authorization grant is invalid"}`)),
				}, nil)

				return mockHTTP, mockStore
			},
			expectErr: errors.New("failed to refresh token: failed to authenticate: invalid_grant - the provided authorization grant is invalid"),
		},
		{
			name: "no credentials available",
			setupMocks: func(ctrl *gomock.Controller) (*httpmock.MockHTTPDoer, *MockCredentialsStore) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockStore := NewMockCredentialsStore(ctrl)

				mockStore.EXPECT().Load().Return(nil, errors.New("no credentials"))

				return mockHTTP, mockStore
			},
			expectErr: errors.New("no valid credentials available"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHTTP, mockStore := tt.setupMocks(ctrl)

			provider := &OIDCProvider{
				TokenEndpoint: "https://example.com/token",
				ClientID:      "test-client-id",
				httpClient:    mockHTTP,
				store:         mockStore,
			}

			creds, err := provider.ensureValidToken(context.Background())

			if tt.expectErr == nil {
				require.NoError(t, err)
				assert.NotNil(t, creds)
				return
			}
			assert.Equal(t, tt.expectErr.Error(), err.Error())
		})
	}
}
