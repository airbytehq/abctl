package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/airbytehq/abctl/internal/airbox"
	airboxmock "github.com/airbytehq/abctl/internal/airbox/mock"
	"github.com/airbytehq/abctl/internal/api"
	apimock "github.com/airbytehq/abctl/internal/api/mock"
	"github.com/airbytehq/abctl/internal/auth"
	internalhttp "github.com/airbytehq/abctl/internal/http"
	httpmock "github.com/airbytehq/abctl/internal/http/mock"
	uimock "github.com/airbytehq/abctl/internal/ui/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestLoginCmd_Run(t *testing.T) {
	tests := []struct {
		name          string
		expectedError string
		setupMocks    func(ctrl *gomock.Controller) (airbox.ConfigStore, internalhttp.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider)
	}{
		{
			name: "OAuth2 success with single org",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, internalhttp.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				// Create config with OAuth2 auth
				config := &airbox.Config{
					CurrentContext: "https://test.airbyte.com",
					Contexts: []airbox.NamedContext{
						{
							Name: "https://test.airbyte.com",
							Context: airbox.Context{
								AirbyteAPIURL: "https://api.test.airbyte.com",
								AirbyteURL:    "https://test.airbyte.com",
								Edition:       "cloud",
								Auth:          airbox.NewAuthWithOAuth2("client-id", "client-secret"),
							},
						},
					},
				}

				// Updated config after org selection
				updatedConfig := *config
				updatedConfig.Contexts[0].Context.OrganizationID = "org-123"
				updatedConfig.Credentials = &auth.Credentials{
					AccessToken: "test-token",
					TokenType:   "Bearer",
					ExpiresAt:   time.Now().Add(time.Hour),
				}

				mockAPI := func(_ context.Context, _ internalhttp.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				orgs := []*api.Organization{
					{ID: "org-123", Name: "Test Org"},
				}

				// UI expectations
				mockUI.EXPECT().Title("Authenticating with Airbyte")
				mockUI.EXPECT().ShowInfo("Connecting to: https://api.test.airbyte.com")
				mockUI.EXPECT().NewLine()
				mockUI.EXPECT().ShowSuccess("Successfully authenticated!")
				mockUI.EXPECT().NewLine()

				// Config store - allow multiple Load/Save calls
				mockStore.EXPECT().Load().Return(config, nil).AnyTimes()
				mockStore.EXPECT().Save(gomock.Any()).Return(nil).AnyTimes()

				// HTTP call for OAuth2 token
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)),
				}, nil)

				// Organization selection
				mockService.EXPECT().ListOrganizations(gomock.Any()).Return(orgs, nil)

				return mockStore, mockHTTP, mockAPI, mockUI
			},
		},
		{
			name: "OAuth2 success with multiple orgs",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, internalhttp.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				config := &airbox.Config{
					CurrentContext: "https://test.airbyte.com",
					Contexts: []airbox.NamedContext{
						{
							Name: "https://test.airbyte.com",
							Context: airbox.Context{
								AirbyteAPIURL: "https://api.test.airbyte.com",
								AirbyteURL:    "https://test.airbyte.com",
								Edition:       "cloud",
								Auth:          airbox.NewAuthWithOAuth2("client-id", "client-secret"),
							},
						},
					},
				}

				mockAPI := func(_ context.Context, _ internalhttp.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				orgs := []*api.Organization{
					{ID: "org-123", Name: "Org A"},
					{ID: "org-456", Name: "Org B"},
				}

				// UI expectations
				mockUI.EXPECT().Title("Authenticating with Airbyte")
				mockUI.EXPECT().ShowInfo("Connecting to: https://api.test.airbyte.com")
				mockUI.EXPECT().NewLine()
				mockUI.EXPECT().ShowSuccess("Successfully authenticated!")
				mockUI.EXPECT().NewLine()
				mockUI.EXPECT().Select("Select an organization:", []string{"Org A", "Org B"}).Return(0, "Org A", nil)

				// Config store - allow multiple Load/Save calls
				mockStore.EXPECT().Load().Return(config, nil).AnyTimes()
				mockStore.EXPECT().Save(gomock.Any()).Return(nil).AnyTimes()

				// HTTP call for OAuth2 token
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)),
				}, nil)

				// Organization selection
				mockService.EXPECT().ListOrganizations(gomock.Any()).Return(orgs, nil)

				return mockStore, mockHTTP, mockAPI, mockUI
			},
		},
		{
			name:          "config load error",
			expectedError: "failed to load airbox config",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, internalhttp.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockAPI := func(_ context.Context, _ internalhttp.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					return nil, nil
				}

				gomock.InOrder(
					mockUI.EXPECT().Title("Authenticating with Airbyte"),
					mockStore.EXPECT().Load().Return(nil, assert.AnError),
				)

				return mockStore, mockHTTP, mockAPI, mockUI
			},
		},
		{
			name:          "config file not found",
			expectedError: "no airbox configuration found",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, internalhttp.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)
				mockAPI := func(_ context.Context, _ internalhttp.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					t.Fatal("API factory should not be called when config load fails")
					return nil, nil
				}

				mockUI.EXPECT().Title("Authenticating with Airbyte")
				mockStore.EXPECT().Load().Return(nil, os.ErrNotExist)

				return mockStore, mockHTTP, mockAPI, mockUI
			},
		},
		{
			name:          "no current context",
			expectedError: "no current context set - please run 'airbox config init' first",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, internalhttp.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				config := &airbox.Config{
					CurrentContext: "", // No current context
					Contexts:       []airbox.NamedContext{},
				}

				mockAPI := func(_ context.Context, _ internalhttp.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					return nil, nil
				}

				gomock.InOrder(
					mockUI.EXPECT().Title("Authenticating with Airbyte"),
					mockStore.EXPECT().Load().Return(config, nil),
				)

				return mockStore, mockHTTP, mockAPI, mockUI
			},
		},
		{
			name:          "API factory error",
			expectedError: "mock api factory error",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, internalhttp.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				config := &airbox.Config{
					CurrentContext: "https://test.airbyte.com",
					Contexts: []airbox.NamedContext{
						{
							Name: "https://test.airbyte.com",
							Context: airbox.Context{
								AirbyteAPIURL: "https://api.test.airbyte.com",
								AirbyteURL:    "https://test.airbyte.com",
								Edition:       "cloud",
								Auth:          airbox.NewAuthWithOAuth2("client-id", "client-secret"),
							},
						},
					},
				}

				mockAPI := func(_ context.Context, _ internalhttp.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					return nil, errors.New("mock api factory error")
				}

				// UI expectations
				mockUI.EXPECT().Title("Authenticating with Airbyte")
				mockUI.EXPECT().ShowInfo("Connecting to: https://api.test.airbyte.com")
				mockUI.EXPECT().NewLine()
				mockUI.EXPECT().ShowSuccess("Successfully authenticated!")
				mockUI.EXPECT().NewLine()

				// Config store - allow multiple Load/Save calls
				mockStore.EXPECT().Load().Return(config, nil).AnyTimes()
				mockStore.EXPECT().Save(gomock.Any()).Return(nil).AnyTimes()

				// HTTP call for OAuth2 token
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)),
				}, nil)

				return mockStore, mockHTTP, mockAPI, mockUI
			},
		},
		{
			name:          "list organizations error",
			expectedError: "assert.AnError general error for testing",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, internalhttp.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				config := &airbox.Config{
					CurrentContext: "https://test.airbyte.com",
					Contexts: []airbox.NamedContext{
						{
							Name: "https://test.airbyte.com",
							Context: airbox.Context{
								AirbyteAPIURL: "https://api.test.airbyte.com",
								AirbyteURL:    "https://test.airbyte.com",
								Edition:       "cloud",
								Auth:          airbox.NewAuthWithOAuth2("client-id", "client-secret"),
							},
						},
					},
				}

				mockAPI := func(_ context.Context, _ internalhttp.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				// UI expectations
				mockUI.EXPECT().Title("Authenticating with Airbyte")
				mockUI.EXPECT().ShowInfo("Connecting to: https://api.test.airbyte.com")
				mockUI.EXPECT().NewLine()
				mockUI.EXPECT().ShowSuccess("Successfully authenticated!")
				mockUI.EXPECT().NewLine()

				// Config store - allow multiple Load/Save calls
				mockStore.EXPECT().Load().Return(config, nil).AnyTimes()
				mockStore.EXPECT().Save(gomock.Any()).Return(nil).AnyTimes()

				// HTTP call for OAuth2 token
				mockHTTP.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)),
				}, nil)

				// Organization selection that fails
				mockService.EXPECT().ListOrganizations(gomock.Any()).Return(nil, assert.AnError)

				return mockStore, mockHTTP, mockAPI, mockUI
			},
		},
		{
			name:          "invalid auth provider type",
			expectedError: "invalid auth configuration - please run 'airbox config init' first",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, internalhttp.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				config := &airbox.Config{
					CurrentContext: "https://test.airbyte.com",
					Contexts: []airbox.NamedContext{
						{
							Name: "https://test.airbyte.com",
							Context: airbox.Context{
								AirbyteAPIURL: "https://api.test.airbyte.com",
								AirbyteURL:    "https://test.airbyte.com",
								Edition:       "enterprise",
								Auth:          airbox.Auth{}, // Invalid/empty auth
							},
						},
					},
				}

				mockAPI := func(_ context.Context, _ internalhttp.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					return nil, nil
				}

				// UI expectations
				mockUI.EXPECT().Title("Authenticating with Airbyte")
				mockUI.EXPECT().ShowInfo("Connecting to: https://api.test.airbyte.com")
				mockUI.EXPECT().NewLine()

				// Config store
				mockStore.EXPECT().Load().Return(config, nil).AnyTimes()

				return mockStore, mockHTTP, mockAPI, mockUI
			},
		},
		{
			name:          "OIDC discovery error",
			expectedError: "failed to discover OIDC provider",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, internalhttp.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				config := &airbox.Config{
					CurrentContext: "https://test.airbyte.com",
					Contexts: []airbox.NamedContext{
						{
							Name: "https://test.airbyte.com",
							Context: airbox.Context{
								AirbyteAPIURL: "https://api.test.airbyte.com",
								AirbyteURL:    "https://test.airbyte.com",
								Edition:       "enterprise",
								Auth:          airbox.NewAuthWithOIDC("https://auth.test.airbyte.com", "oidc-client-id"),
							},
						},
					},
				}

				mockAPI := func(_ context.Context, _ internalhttp.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					return nil, nil
				}

				// UI expectations
				mockUI.EXPECT().Title("Authenticating with Airbyte")
				mockUI.EXPECT().ShowInfo("Connecting to: https://api.test.airbyte.com")
				mockUI.EXPECT().NewLine()
				mockUI.EXPECT().ShowInfo("Discovering OIDC provider at: https://auth.test.airbyte.com")

				// Config store
				mockStore.EXPECT().Load().Return(config, nil).AnyTimes()

				// OIDC discovery fails
				mockHTTP.EXPECT().Do(gomock.Any()).Return(nil, assert.AnError)

				return mockStore, mockHTTP, mockAPI, mockUI
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore, mockHTTP, mockAPI, mockUI := tt.setupMocks(ctrl)

			cmd := &LoginCmd{}
			err := cmd.Run(context.Background(), mockStore, mockHTTP, mockAPI, mockUI)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
