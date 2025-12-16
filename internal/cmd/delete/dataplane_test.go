package delete

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/airbytehq/abctl/internal/airbox"
	airboxmock "github.com/airbytehq/abctl/internal/airbox/mock"
	"github.com/airbytehq/abctl/internal/api"
	apimock "github.com/airbytehq/abctl/internal/api/mock"
	"github.com/airbytehq/abctl/internal/http"
	httpmock "github.com/airbytehq/abctl/internal/http/mock"
	uimock "github.com/airbytehq/abctl/internal/ui/mock"
)

func TestDataplaneCmd_Run(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		force         bool
		expectedError string
		setupMocks    func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider)
	}{
		{
			name:  "success",
			id:    "dp-123",
			force: true,
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockCfg := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockAPI := func(_ context.Context, _ http.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				gomock.InOrder(
					mockUI.EXPECT().Title("Deleting dataplane"),
					mockService.EXPECT().DeleteDataplane(gomock.Any(), "dp-123").Return(nil),
					mockUI.EXPECT().ShowSuccess("Dataplane ID 'dp-123' deleted successfully"),
					mockUI.EXPECT().NewLine(),
				)

				return mockCfg, mockHTTP, mockAPI, mockUI
			},
		},
		{
			name:          "factory error",
			id:            "dp-123",
			force:         true,
			expectedError: "mock api factory error",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockCfg := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockAPI := func(_ context.Context, _ http.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					return nil, errors.New("mock api factory error")
				}

				gomock.InOrder(
					mockUI.EXPECT().Title("Deleting dataplane"),
				)

				return mockCfg, mockHTTP, mockAPI, mockUI
			},
		},
		{
			name:          "delete error",
			id:            "dp-123",
			force:         true,
			expectedError: "failed to delete dataplane",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockCfg := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockAPI := func(_ context.Context, _ http.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				gomock.InOrder(
					mockUI.EXPECT().Title("Deleting dataplane"),
					mockService.EXPECT().DeleteDataplane(gomock.Any(), "dp-123").Return(assert.AnError),
				)

				return mockCfg, mockHTTP, mockAPI, mockUI
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cfg, mockHTTP, mockAPI, mockUI := tt.setupMocks(ctrl)

			cmd := &DataplaneCmd{
				ID:    tt.id,
				Force: tt.force,
			}

			err := cmd.Run(context.Background(), cfg, mockHTTP, mockAPI, mockUI)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}