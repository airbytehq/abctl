package get

import (
	"context"
	"errors"
	"testing"

	"github.com/airbytehq/abctl/internal/airbox"
	airboxmock "github.com/airbytehq/abctl/internal/airbox/mock"
	"github.com/airbytehq/abctl/internal/api"
	apimock "github.com/airbytehq/abctl/internal/api/mock"
	"github.com/airbytehq/abctl/internal/http"
	httpmock "github.com/airbytehq/abctl/internal/http/mock"
	uimock "github.com/airbytehq/abctl/internal/ui/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDataplaneCmd_Run(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		expectedError string
		setupMocks    func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider)
	}{
		{
			name: "list success",
			id:   "",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockCfg := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				dataplanes := []api.Dataplane{
					{DataplaneID: "dp-1", Name: "test-dataplane-1", RegionID: "region-1"},
					{DataplaneID: "dp-2", Name: "test-dataplane-2", RegionID: "region-2"},
				}

				mockAPI := func(_ context.Context, _ http.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				gomock.InOrder(
					mockService.EXPECT().ListDataplanes(gomock.Any()).Return(dataplanes, nil),
					mockUI.EXPECT().ShowJSON(dataplanes).Return(nil),
				)

				return mockCfg, mockHTTP, mockAPI, mockUI
			},
		},
		{
			name: "get success",
			id:   "dp-1",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockCfg := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				dataplane := &api.Dataplane{
					DataplaneID: "dp-1",
					Name:        "test-dataplane",
					RegionID:    "region-1",
				}

				mockAPI := func(_ context.Context, _ http.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				gomock.InOrder(
					mockService.EXPECT().GetDataplane(gomock.Any(), "dp-1").Return(dataplane, nil),
					mockUI.EXPECT().ShowJSON(dataplane).Return(nil),
				)

				return mockCfg, mockHTTP, mockAPI, mockUI
			},
		},
		{
			name:          "factory error",
			id:            "dp-1",
			expectedError: "mock api factory error",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockCfg := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockAPI := func(_ context.Context, _ http.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					return nil, errors.New("mock api factory error")
				}

				return mockCfg, mockHTTP, mockAPI, mockUI
			},
		},
		{
			name:          "get dataplane error",
			id:            "dp-1",
			expectedError: "assert.AnError general error for testing",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockCfg := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockAPI := func(_ context.Context, _ http.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				gomock.InOrder(
					mockService.EXPECT().GetDataplane(gomock.Any(), "dp-1").Return(nil, assert.AnError),
				)

				return mockCfg, mockHTTP, mockAPI, mockUI
			},
		},
		{
			name:          "list dataplanes error",
			id:            "",
			expectedError: "failed to list dataplanes",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockCfg := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockAPI := func(_ context.Context, _ http.HTTPDoer, _ airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				gomock.InOrder(
					mockService.EXPECT().ListDataplanes(gomock.Any()).Return(nil, assert.AnError),
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
				ID: tt.id,
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
