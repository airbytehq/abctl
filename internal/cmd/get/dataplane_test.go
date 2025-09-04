package get

import (
	"context"
	"io"
	stdhttp "net/http"
	"strings"
	"testing"

	"github.com/airbytehq/abctl/internal/airbox"
	airboxmock "github.com/airbytehq/abctl/internal/airbox/mock"
	"github.com/airbytehq/abctl/internal/api"
	"github.com/airbytehq/abctl/internal/http"
	httpmock "github.com/airbytehq/abctl/internal/http/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDataplaneCmd_Run(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		expectedError string
		setupMocks    func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider)
	}{
		{
			name: "list success",
			id:   "",
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider) {
				mockCfg := airboxmock.NewMockConfigProvider(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)

				mockCfg.EXPECT().Load().Return(&airbox.Config{
					Credentials: &airbox.Credentials{AccessToken: "valid-token"},
				}, nil).Times(1)

				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(`[
						{
							"dataplaneId": "dp-1",
							"name": "test-dataplane-1",
							"regionId": "region-1",
							"enabled": true
						},
						{
							"dataplaneId": "dp-2",
							"name": "test-dataplane-2",
							"regionId": "region-2",
							"enabled": true
						}
					]`)),
					Header: make(stdhttp.Header),
				}, nil).Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return api.NewClient(mockHTTP), nil
				}

				return mockHTTP, mockAPI, mockCfg
			},
		},
		{
			name: "success",
			id:   "dp-1",
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider) {
				mockCfg := airboxmock.NewMockConfigProvider(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)

				mockCfg.EXPECT().Load().Return(&airbox.Config{
					Credentials: &airbox.Credentials{AccessToken: "valid-token"},
				}, nil).Times(1)

				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(`{
						"dataplaneId": "dp-1",
						"name": "test-dataplane",
						"regionId": "region-1",
						"enabled": true
					}`)),
					Header: make(stdhttp.Header),
				}, nil).Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return api.NewClient(mockHTTP), nil
				}

				return mockHTTP, mockAPI, mockCfg
			},
		},
		{
			name:          "not authenticated",
			id:            "dp-1",
			expectedError: "not authenticated - please run 'airbox auth login' first",
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider) {
				mockCfg := airboxmock.NewMockConfigProvider(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)

				mockCfg.EXPECT().Load().Return(&airbox.Config{
					Credentials: nil, // No credentials
				}, nil).Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return api.NewClient(nil), nil
				}

				return mockHTTP, mockAPI, mockCfg
			},
		},
		{
			name:          "config load error",
			id:            "dp-1",
			expectedError: "failed to load config",
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider) {
				mockCfg := airboxmock.NewMockConfigProvider(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)

				mockCfg.EXPECT().Load().Return(nil, assert.AnError).Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return api.NewClient(nil), nil
				}

				return mockHTTP, mockAPI, mockCfg
			},
		},
		{
			name:          "factory error",
			id:            "dp-1",
			expectedError: "failed to create API client",
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider) {
				mockCfg := airboxmock.NewMockConfigProvider(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)

				mockCfg.EXPECT().Load().Return(&airbox.Config{
					Credentials: &airbox.Credentials{AccessToken: "valid-token"},
				}, nil).Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return nil, assert.AnError
				}

				return mockHTTP, mockAPI, mockCfg
			},
		},
		{
			name:          "API error",
			id:            "non-existent",
			expectedError: "API error 404",
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider) {
				mockCfg := airboxmock.NewMockConfigProvider(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)

				mockCfg.EXPECT().Load().Return(&airbox.Config{
					Credentials: &airbox.Credentials{AccessToken: "valid-token"},
				}, nil).Times(1)

				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 404,
					Body:       io.NopCloser(strings.NewReader(`{"error": "Dataplane not found"}`)),
					Header:     make(stdhttp.Header),
				}, nil).Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return api.NewClient(mockHTTP), nil
				}

				return mockHTTP, mockAPI, mockCfg
			},
		},
		{
			name:          "list API error",
			id:            "",
			expectedError: "failed to list dataplanes",
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider) {
				mockCfg := airboxmock.NewMockConfigProvider(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)

				mockCfg.EXPECT().Load().Return(&airbox.Config{
					Credentials: &airbox.Credentials{AccessToken: "valid-token"},
				}, nil).Times(1)

				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 500,
					Body:       io.NopCloser(strings.NewReader(`{"error": "Internal server error"}`)),
					Header:     make(stdhttp.Header),
				}, nil).Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return api.NewClient(mockHTTP), nil
				}

				return mockHTTP, mockAPI, mockCfg
			},
		},
		{
			name:          "HTTP error",
			id:            "dp-1",
			expectedError: "assert.AnError general error for testing",
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider) {
				mockCfg := airboxmock.NewMockConfigProvider(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)

				mockCfg.EXPECT().Load().Return(&airbox.Config{
					Credentials: &airbox.Credentials{AccessToken: "valid-token"},
				}, nil).Times(1)

				mockHTTP.EXPECT().Do(gomock.Any()).Return(nil, assert.AnError).Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return api.NewClient(mockHTTP), nil
				}

				return mockHTTP, mockAPI, mockCfg
			},
		},
		{
			name:          "list HTTP error",
			id:            "",
			expectedError: "failed to list dataplanes",
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider) {
				mockCfg := airboxmock.NewMockConfigProvider(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)

				mockCfg.EXPECT().Load().Return(&airbox.Config{
					Credentials: &airbox.Credentials{AccessToken: "valid-token"},
				}, nil).Times(1)

				mockHTTP.EXPECT().Do(gomock.Any()).Return(nil, assert.AnError).Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return api.NewClient(mockHTTP), nil
				}

				return mockHTTP, mockAPI, mockCfg
			},
		},
		{
			name:          "invalid JSON",
			id:            "dp-1",
			expectedError: "invalid character",
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider) {
				mockCfg := airboxmock.NewMockConfigProvider(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)

				mockCfg.EXPECT().Load().Return(&airbox.Config{
					Credentials: &airbox.Credentials{AccessToken: "valid-token"},
				}, nil).Times(1)

				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{invalid json}`)),
					Header:     make(stdhttp.Header),
				}, nil).Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return api.NewClient(mockHTTP), nil
				}

				return mockHTTP, mockAPI, mockCfg
			},
		},
		{
			name:          "list invalid JSON",
			id:            "",
			expectedError: "invalid character",
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider) {
				mockCfg := airboxmock.NewMockConfigProvider(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)

				mockCfg.EXPECT().Load().Return(&airbox.Config{
					Credentials: &airbox.Credentials{AccessToken: "valid-token"},
				}, nil).Times(1)

				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`[{invalid json}]`)),
					Header:     make(stdhttp.Header),
				}, nil).Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return api.NewClient(mockHTTP), nil
				}

				return mockHTTP, mockAPI, mockCfg
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHTTP, mockAPI, cfg := tt.setupMocks(ctrl)

			cmd := &DataplaneCmd{
				ID: tt.id,
			}

			err := cmd.Run(context.Background(), mockHTTP, mockAPI, cfg)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
