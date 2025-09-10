package auth

import (
	"context"
	"errors"
	"fmt"
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

func TestSwitchOrganizationCmd_Run(t *testing.T) {
	tests := []struct {
		name          string
		expectedError string
		setupMocks    func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider)
	}{
		{
			name: "success with multiple orgs",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
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
								OrganizationID: "current-org",
							},
						},
					},
				}

				orgs := []*api.Organization{
					{ID: "org-1", Name: "Org A"},
					{ID: "org-2", Name: "Org B"},
				}

				mockAPIFactory := func(ctx context.Context, httpClient http.HTTPDoer, cfg airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				// UI expectations
				mockUI.EXPECT().Title("Switching Workspace")
				mockUI.EXPECT().ShowInfo("Switch between your user's organizations.")
				mockUI.EXPECT().NewLine()
				mockUI.EXPECT().Select("Select an organization:", []string{"Org A", "Org B"}).Return(1, "Org B", nil)
				mockUI.EXPECT().ShowSuccess("Organization switched successfully!")
				mockUI.EXPECT().NewLine()

				// Config and API expectations
				mockStore.EXPECT().Load().Return(config, nil)
				mockService.EXPECT().ListOrganizations(gomock.Any()).Return(orgs, nil)
				mockStore.EXPECT().Save(gomock.Any()).Return(nil).Times(2)

				return mockStore, mockHTTP, mockAPIFactory, mockUI
			},
		},
		{
			name:          "API factory error",
			expectedError: "mock api factory error",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockAPIFactory := func(ctx context.Context, httpClient http.HTTPDoer, cfg airbox.ConfigStore) (api.Service, error) {
					return nil, errors.New("mock api factory error")
				}

				mockUI.EXPECT().Title("Switching Workspace")
				mockUI.EXPECT().ShowInfo("Switch between your user's organizations.")
				mockUI.EXPECT().NewLine()

				return mockStore, mockHTTP, mockAPIFactory, mockUI
			},
		},
		{
			name:          "config load error",
			expectedError: "failed to load config",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockAPIFactory := func(ctx context.Context, httpClient http.HTTPDoer, cfg airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				mockUI.EXPECT().Title("Switching Workspace")
				mockUI.EXPECT().ShowInfo("Switch between your user's organizations.")
				mockUI.EXPECT().NewLine()
				mockStore.EXPECT().Load().Return(nil, assert.AnError)

				return mockStore, mockHTTP, mockAPIFactory, mockUI
			},
		},
		{
			name:          "no organizations found",
			expectedError: "no organizations found",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				config := &airbox.Config{
					CurrentContext: "https://test.airbyte.com",
					Contexts: []airbox.NamedContext{
						{
							Name:    "https://test.airbyte.com",
							Context: airbox.Context{},
						},
					},
				}

				mockAPIFactory := func(ctx context.Context, httpClient http.HTTPDoer, cfg airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				mockUI.EXPECT().Title("Switching Workspace")
				mockUI.EXPECT().ShowInfo("Switch between your user's organizations.")
				mockUI.EXPECT().NewLine()
				mockStore.EXPECT().Load().Return(config, nil)
				mockService.EXPECT().ListOrganizations(gomock.Any()).Return([]*api.Organization{}, nil)

				return mockStore, mockHTTP, mockAPIFactory, mockUI
			},
		},
		{
			name: "single organization auto-select",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				config := &airbox.Config{
					CurrentContext: "https://test.airbyte.com",
					Contexts: []airbox.NamedContext{
						{
							Name:    "https://test.airbyte.com",
							Context: airbox.Context{},
						},
					},
				}

				orgs := []*api.Organization{
					{ID: "org-1", Name: "Only Org"},
				}

				mockAPIFactory := func(ctx context.Context, httpClient http.HTTPDoer, cfg airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				mockUI.EXPECT().Title("Switching Workspace")
				mockUI.EXPECT().ShowInfo("Switch between your user's organizations.")
				mockUI.EXPECT().NewLine()
				mockUI.EXPECT().ShowInfo("Setting organization to: Only Org")
				mockUI.EXPECT().NewLine()
				mockUI.EXPECT().ShowSuccess("Organization switched successfully!")
				mockUI.EXPECT().NewLine()
				mockStore.EXPECT().Load().Return(config, nil)
				mockService.EXPECT().ListOrganizations(gomock.Any()).Return(orgs, nil)
				mockStore.EXPECT().Save(gomock.Any()).Return(nil).Times(2)

				return mockStore, mockHTTP, mockAPIFactory, mockUI
			},
		},
		{
			name: "success with many orgs (filterable select)",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, *uimock.MockProvider) {
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
								OrganizationID: "current-org",
							},
						},
					},
				}

				// Create 15 orgs to trigger FilterableSelect (>10)
				orgs := make([]*api.Organization, 15)
				orgNames := make([]string, 15)
				for i := range 15 {
					orgs[i] = &api.Organization{ID: fmt.Sprintf("org-%d", i), Name: fmt.Sprintf("Organization %d", i)}
					orgNames[i] = fmt.Sprintf("Organization %d", i)
				}

				mockAPIFactory := func(ctx context.Context, httpClient http.HTTPDoer, cfg airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				// UI expectations
				mockUI.EXPECT().Title("Switching Workspace")
				mockUI.EXPECT().ShowInfo("Switch between your user's organizations.")
				mockUI.EXPECT().NewLine()
				mockUI.EXPECT().FilterableSelect("Select an organization", orgNames).Return(5, "Organization 5", nil)
				mockUI.EXPECT().ShowSuccess("Organization switched successfully!")
				mockUI.EXPECT().NewLine()

				// Config and API expectations
				mockStore.EXPECT().Load().Return(config, nil)
				mockService.EXPECT().ListOrganizations(gomock.Any()).Return(orgs, nil)
				mockStore.EXPECT().Save(gomock.Any()).Return(nil).Times(2)

				return mockStore, mockHTTP, mockAPIFactory, mockUI
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore, mockHTTP, mockAPI, mockUI := tt.setupMocks(ctrl)

			cmd := &SwitchOrganizationCmd{}
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
