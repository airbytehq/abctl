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

func TestNewOAuth2Provider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTPClient := httpmock.NewMockHTTPDoer(ctrl)
	mockStore := NewMockCredentialsStore(ctrl)

	// Initial token fetch during creation
	mockStore.EXPECT().Load().Return(nil, nil)
	mockHTTPClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"access_token": "test-token", "expires_in": 3600}`)),
	}, nil)
	mockStore.EXPECT().Save(gomock.Any()).Return(nil)

	provider, err := NewOAuth2Provider(context.Background(), "test-client-id", "test-client-secret", mockHTTPClient, mockStore)

	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "test-client-id", provider.ClientID)
	assert.Equal(t, "test-client-secret", provider.ClientSecret)
	assert.Equal(t, "/v1/applications/token", provider.TokenEndpoint)
}

func TestOAuth2Provider_GetToken(t *testing.T) {
	tests := []struct {
		name      string
		setupHTTP func(mock *httpmock.MockHTTPDoer)
		expectErr error
	}{
		{
			name: "success",
			setupHTTP: func(mock *httpmock.MockHTTPDoer) {
				response := `{
					"access_token": "oauth2-access-token",
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
			name: "http error",
			setupHTTP: func(mock *httpmock.MockHTTPDoer) {
				mock.EXPECT().Do(gomock.Any()).Return(nil, errors.New("network error"))
			},
			expectErr: errors.New("failed to authenticate with OAuth2 credentials: failed to get fresh token: failed to execute token request: network error"),
		},
		{
			name: "oauth error response",
			setupHTTP: func(mock *httpmock.MockHTTPDoer) {
				response := `{
					"error": "invalid_client",
					"error_description": "Client authentication failed"
				}`
				mock.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 401,
					Body:       io.NopCloser(strings.NewReader(response)),
				}, nil)
			},
			expectErr: errors.New("failed to authenticate with OAuth2 credentials: failed to get fresh token: failed to authenticate: invalid_client - client authentication failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHTTPClient := httpmock.NewMockHTTPDoer(ctrl)
			tt.setupHTTP(mockHTTPClient)

			mockStore := NewMockCredentialsStore(ctrl)
			mockStore.EXPECT().Load().Return(nil, nil).AnyTimes()
			mockStore.EXPECT().Save(gomock.Any()).Return(nil).AnyTimes()

			provider, err := NewOAuth2Provider(context.Background(), "test-client-id", "test-client-secret", mockHTTPClient, mockStore)

			if tt.expectErr == nil {
				require.NoError(t, err)
				assert.NotNil(t, provider)
				return
			}
			assert.Equal(t, tt.expectErr.Error(), err.Error())
			assert.Nil(t, provider)
		})
	}
}

func TestOAuth2Provider_RefreshToken(t *testing.T) {
	tests := []struct {
		name      string
		setupHTTP func(mock *httpmock.MockHTTPDoer)
		expectErr error
	}{
		{
			name: "success",
			setupHTTP: func(mock *httpmock.MockHTTPDoer) {
				response := `{
					"access_token": "refreshed-access-token",
					"refresh_token": "new-refresh-token",
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
			name: "invalid refresh token",
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

			// Create provider with minimal setup for testing RefreshToken
			provider := &OAuth2ClientCredentialsProvider{
				TokenEndpoint: "/v1/applications/token",
				ClientID:      "test-client-id",
				ClientSecret:  "test-client-secret",
			}
			tokenResp, err := provider.RefreshToken(context.Background(), "old-refresh-token", mockHTTPClient)

			if tt.expectErr == nil {
				require.NoError(t, err)
				assert.NotNil(t, tokenResp)
				return
			}
			assert.Equal(t, tt.expectErr.Error(), err.Error())
		})
	}
}

func TestOAuth2Provider_GetTokenEndpoint(t *testing.T) {
	provider := &OAuth2ClientCredentialsProvider{
		TokenEndpoint: "/v1/applications/token",
		ClientID:      "test-client-id",
		ClientSecret:  "test-client-secret",
	}
	endpoint := provider.GetTokenEndpoint()

	assert.Equal(t, "/v1/applications/token", endpoint)
}

func TestOAuth2Provider_AuthEndpointHandler(t *testing.T) {
	provider := &OAuth2ClientCredentialsProvider{
		TokenEndpoint: "/v1/applications/token",
		ClientID:      "test-client-id",
		ClientSecret:  "test-client-secret",
	}
	handler := provider.AuthEndpointHandler()

	assert.Nil(t, handler)
}

func TestOAuth2Provider_Load(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockCredentialsStore(ctrl)
	provider := &OAuth2ClientCredentialsProvider{
		TokenEndpoint: "/v1/applications/token",
		ClientID:      "test-client-id",
		ClientSecret:  "test-client-secret",
		store:         mockStore,
	}

	expectedCreds := &Credentials{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}
	mockStore.EXPECT().Load().Return(expectedCreds, nil)

	creds, err := provider.Load()

	assert.NoError(t, err)
	assert.Equal(t, expectedCreds, creds)
}

func TestOAuth2Provider_Save(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockCredentialsStore(ctrl)
	provider := &OAuth2ClientCredentialsProvider{
		store: mockStore,
	}

	creds := &Credentials{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}
	mockStore.EXPECT().Save(creds).Return(nil)

	err := provider.Save(creds)

	assert.NoError(t, err)
}

func TestOAuth2Provider_Do(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(ctrl *gomock.Controller) (*httpmock.MockHTTPDoer, *MockCredentialsStore)
		expectErr   error
		expectCalls int
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
			expectCalls: 1,
		},
		{
			name: "retry on 401 with fresh token",
			setupMocks: func(ctrl *gomock.Controller) (*httpmock.MockHTTPDoer, *MockCredentialsStore) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockStore := NewMockCredentialsStore(ctrl)

				// First call - return valid token
				mockStore.EXPECT().Load().Return(&Credentials{
					AccessToken: "expired-token",
					TokenType:   "Bearer",
					ExpiresAt:   time.Now().Add(time.Hour),
				}, nil)

				// First HTTP request returns 401
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 401,
					Body:       io.NopCloser(strings.NewReader("unauthorized")),
				}, nil)

				// Mark token as expired and get fresh one
				mockStore.EXPECT().Save(gomock.Any()).Return(nil)
				mockStore.EXPECT().Load().Return(nil, nil)
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"access_token": "fresh-token", "expires_in": 3600}`)),
				}, nil)
				mockStore.EXPECT().Save(gomock.Any()).Return(nil)

				// Second HTTP request with fresh token succeeds
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("success")),
				}, nil)

				return mockHTTP, mockStore
			},
			expectCalls: 3,
		},
		{
			name: "error getting valid token",
			setupMocks: func(ctrl *gomock.Controller) (*httpmock.MockHTTPDoer, *MockCredentialsStore) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockStore := NewMockCredentialsStore(ctrl)

				mockStore.EXPECT().Load().Return(nil, errors.New("store error"))
				mockHTTP.EXPECT().Do(gomock.Any()).Return(nil, errors.New("network error"))

				return mockHTTP, mockStore
			},
			expectErr: errors.New("failed to get valid token: failed to get fresh token: failed to execute token request: network error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHTTP, mockStore := tt.setupMocks(ctrl)

			provider := &OAuth2ClientCredentialsProvider{
				TokenEndpoint: "/v1/applications/token",
				ClientID:      "test-client-id",
				ClientSecret:  "test-client-secret",
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
