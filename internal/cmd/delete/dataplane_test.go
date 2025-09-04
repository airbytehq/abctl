package delete

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
	"github.com/airbytehq/abctl/internal/ui"
	uimock "github.com/airbytehq/abctl/internal/ui/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDataplaneCmd_Run(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		force         bool
		expectedError string
		setupMocks    func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider, ui.Provider)
	}{
		{
			name:  "success",
			id:    "dp-123",
			force: true,
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider, ui.Provider) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockConfig := airboxmock.NewMockConfigProvider(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockConfig.EXPECT().Load().Return(&airbox.Config{
					Credentials: &airbox.Credentials{AccessToken: "valid-token"},
				}, nil).Times(1)

				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 204,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(stdhttp.Header),
				}, nil).Times(1)

				mockUI.EXPECT().Title("Deleting dataplane").Times(1)
				mockUI.EXPECT().ShowSuccess("Dataplane ID 'dp-123' deleted successfully").Times(1)
				mockUI.EXPECT().NewLine().Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return api.NewClient(mockHTTP), nil
				}

				return mockHTTP, mockAPI, mockConfig, mockUI
			},
		},
		{
			name:  "namespace resolution",
			id:    "dp-123",
			force: true,
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider, ui.Provider) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockConfig := airboxmock.NewMockConfigProvider(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockConfig.EXPECT().Load().Return(&airbox.Config{
					Credentials: &airbox.Credentials{AccessToken: "valid-token"},
				}, nil).Times(1)

				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 204,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(stdhttp.Header),
				}, nil).Times(1)

				mockUI.EXPECT().Title("Deleting dataplane").Times(1)
				mockUI.EXPECT().ShowSuccess("Dataplane ID 'dp-123' deleted successfully").Times(1)
				mockUI.EXPECT().NewLine().Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return api.NewClient(mockHTTP), nil
				}

				return mockHTTP, mockAPI, mockConfig, mockUI
			},
		},
		{
			name:          "not authenticated",
			id:            "dp-123",
			force:         true,
			expectedError: "not authenticated - please run 'airbox auth login' first",
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider, ui.Provider) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockConfig := airboxmock.NewMockConfigProvider(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockConfig.EXPECT().Load().Return(&airbox.Config{
					Credentials: nil, // No credentials
				}, nil).Times(1)

				mockUI.EXPECT().Title("Deleting dataplane").Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return api.NewClient(mockHTTP), nil
				}

				return mockHTTP, mockAPI, mockConfig, mockUI
			},
		},
		{
			name:          "config load error",
			id:            "dp-123",
			force:         true,
			expectedError: "failed to load config",
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider, ui.Provider) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockConfig := airboxmock.NewMockConfigProvider(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockConfig.EXPECT().Load().Return(nil, assert.AnError).Times(1)

				mockUI.EXPECT().Title("Deleting dataplane").Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return api.NewClient(mockHTTP), nil
				}

				return mockHTTP, mockAPI, mockConfig, mockUI
			},
		},
		{
			name:          "factory error",
			id:            "dp-123",
			force:         true,
			expectedError: "failed to create API client",
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider, ui.Provider) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockConfig := airboxmock.NewMockConfigProvider(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockConfig.EXPECT().Load().Return(&airbox.Config{
					Credentials: &airbox.Credentials{AccessToken: "valid-token"},
				}, nil).Times(1)

				mockUI.EXPECT().Title("Deleting dataplane").Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return nil, assert.AnError
				}

				return mockHTTP, mockAPI, mockConfig, mockUI
			},
		},
		{
			name:          "API error",
			id:            "dp-nonexistent",
			force:         true,
			expectedError: "failed to delete dataplane",
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider, ui.Provider) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockConfig := airboxmock.NewMockConfigProvider(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockConfig.EXPECT().Load().Return(&airbox.Config{
					Credentials: &airbox.Credentials{AccessToken: "valid-token"},
				}, nil).Times(1)

				mockHTTP.EXPECT().Do(gomock.Any()).Return(&stdhttp.Response{
					StatusCode: 404,
					Body:       io.NopCloser(strings.NewReader(`{"error": "Dataplane not found"}`)),
					Header:     make(stdhttp.Header),
				}, nil).Times(1)

				mockUI.EXPECT().Title("Deleting dataplane").Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return api.NewClient(mockHTTP), nil
				}

				return mockHTTP, mockAPI, mockConfig, mockUI
			},
		},
		{
			name:          "HTTP error",
			id:            "dp-123",
			force:         true,
			expectedError: "assert.AnError general error for testing",
			setupMocks: func(ctrl *gomock.Controller) (http.HTTPDoer, api.Factory, airbox.ConfigProvider, ui.Provider) {
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockConfig := airboxmock.NewMockConfigProvider(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockConfig.EXPECT().Load().Return(&airbox.Config{
					Credentials: &airbox.Credentials{AccessToken: "valid-token"},
				}, nil).Times(1)

				mockHTTP.EXPECT().Do(gomock.Any()).Return(nil, assert.AnError).Times(1)

				mockUI.EXPECT().Title("Deleting dataplane").Times(1)

				mockAPI := func(_ context.Context, _ airbox.ConfigProvider, _ http.HTTPDoer) (*api.Client, error) {
					return api.NewClient(mockHTTP), nil
				}

				return mockHTTP, mockAPI, mockConfig, mockUI
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHTTP, mockAPI, mockCfg, mockUI := tt.setupMocks(ctrl)

			cmd := &DataplaneCmd{
				ID:    tt.id,
				Force: tt.force,
			}

			err := cmd.Run(context.Background(), mockHTTP, mockAPI, mockCfg, mockUI)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
