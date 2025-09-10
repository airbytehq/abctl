package install

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	goHelm "github.com/mittwald/go-helm-client"

	"github.com/airbytehq/abctl/internal/airbox"
	airboxmock "github.com/airbytehq/abctl/internal/airbox/mock"
	"github.com/airbytehq/abctl/internal/api"
	apimock "github.com/airbytehq/abctl/internal/api/mock"
	"github.com/airbytehq/abctl/internal/helm"
	helmmock "github.com/airbytehq/abctl/internal/helm/mock"
	"github.com/airbytehq/abctl/internal/http"
	httpmock "github.com/airbytehq/abctl/internal/http/mock"
	"github.com/airbytehq/abctl/internal/k8s"
	k8smock "github.com/airbytehq/abctl/internal/k8s/mock"
	uimock "github.com/airbytehq/abctl/internal/ui/mock"
)

func TestDataplaneCmd_Run(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		withKind      bool
		expectedError string
		setupMocks    func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, helm.Factory, k8s.ClusterFactory, *uimock.MockProvider)
	}{
		{
			name:          "API factory error",
			namespace:     "test-namespace",
			expectedError: "mock api factory error",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, helm.Factory, k8s.ClusterFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)
				
				mockHelmFactory := func(kubeConfig, kubeContext, namespace string) (goHelm.Client, error) {
					return nil, nil
				}

				mockAPIFactory := func(ctx context.Context, httpClient http.HTTPDoer, cfg airbox.ConfigStore) (api.Service, error) {
					return nil, errors.New("mock api factory error")
				}

				mockClusterFactory := func(ctx context.Context, clusterName string) (k8s.Cluster, error) {
					return k8smock.NewMockCluster(ctrl), nil
				}

				mockUI.EXPECT().Title("Starting interactive dataplane installation")

				return mockStore, mockHTTP, mockAPIFactory, mockHelmFactory, mockClusterFactory, mockUI
			},
		},
		{
			name:          "config load error",
			namespace:     "test-namespace",
			expectedError: "failed to load config",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, helm.Factory, k8s.ClusterFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)
				
				mockHelmFactory := func(kubeConfig, kubeContext, namespace string) (goHelm.Client, error) {
					return nil, nil
				}

				mockAPIFactory := func(ctx context.Context, httpClient http.HTTPDoer, cfg airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				mockClusterFactory := func(ctx context.Context, clusterName string) (k8s.Cluster, error) {
					return k8smock.NewMockCluster(ctrl), nil
				}

				mockUI.EXPECT().Title("Starting interactive dataplane installation")
				mockStore.EXPECT().Load().Return(nil, assert.AnError)

				return mockStore, mockHTTP, mockAPIFactory, mockHelmFactory, mockClusterFactory, mockUI
			},
		},
		{
			name:          "no organization context",
			namespace:     "test-namespace",
			expectedError: "no organization set in context",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, helm.Factory, k8s.ClusterFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)
				
				mockHelmFactory := func(kubeConfig, kubeContext, namespace string) (goHelm.Client, error) {
					return nil, nil
				}

				config := &airbox.Config{
					CurrentContext: "https://test.airbyte.com",
					Contexts: []airbox.NamedContext{
						{
							Name: "https://test.airbyte.com",
							Context: airbox.Context{
								OrganizationID: "", // No org ID
							},
						},
					},
				}

				mockAPIFactory := func(ctx context.Context, httpClient http.HTTPDoer, cfg airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				mockClusterFactory := func(ctx context.Context, clusterName string) (k8s.Cluster, error) {
					return k8smock.NewMockCluster(ctrl), nil
				}

				mockUI.EXPECT().Title("Starting interactive dataplane installation")
				mockStore.EXPECT().Load().Return(config, nil)

				return mockStore, mockHTTP, mockAPIFactory, mockHelmFactory, mockClusterFactory, mockUI
			},
		},
		{
			name:          "helm client creation error",
			namespace:     "test-namespace",
			expectedError: "failed to create helm client",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, helm.Factory, k8s.ClusterFactory, *uimock.MockProvider) {
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
								OrganizationID: "test-org-id",
							},
						},
					},
				}

				mockAPIFactory := func(ctx context.Context, httpClient http.HTTPDoer, cfg airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				mockHelmFactory := func(kubeConfig, kubeContext, namespace string) (goHelm.Client, error) {
					return nil, assert.AnError
				}

				mockClusterFactory := func(ctx context.Context, clusterName string) (k8s.Cluster, error) {
					return k8smock.NewMockCluster(ctrl), nil
				}

				mockUI.EXPECT().Title("Starting interactive dataplane installation")
				mockStore.EXPECT().Load().Return(config, nil)
				mockService.EXPECT().ListRegions(gomock.Any(), "test-org-id").Return([]*api.Region{
					{ID: "region-1", Name: "Test Region", Location: "US", CloudProvider: "AWS"},
				}, nil)
				mockUI.EXPECT().Select("Select region option:", []string{"Use existing region", "Create new region"}).Return(0, "Use existing region", nil)
				mockUI.EXPECT().FilterableSelect("Select existing region:", []string{"Test Region - US (AWS)"}).Return(0, "Test Region - US (AWS)", nil)
				mockUI.EXPECT().TextInput("Enter dataplane name:", "my-dataplane", gomock.Any()).Return("test-dataplane", nil)

				return mockStore, mockHTTP, mockAPIFactory, mockHelmFactory, mockClusterFactory, mockUI
			},
		},
		{
			name:      "success flow without Kind cluster",
			namespace: "test-namespace",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, helm.Factory, k8s.ClusterFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)
				mockHelmClient := helmmock.NewMockClient(ctrl)

				config := &airbox.Config{
					CurrentContext: "https://test.airbyte.com",
					Contexts: []airbox.NamedContext{
						{
							Name: "https://test.airbyte.com",
							Context: airbox.Context{
								OrganizationID: "test-org-id",
							},
						},
					},
				}

				mockAPIFactory := func(ctx context.Context, httpClient http.HTTPDoer, cfg airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				mockHelmFactory := func(kubeConfig, kubeContext, namespace string) (goHelm.Client, error) {
					return mockHelmClient, nil
				}

				mockClusterFactory := func(ctx context.Context, clusterName string) (k8s.Cluster, error) {
					return k8smock.NewMockCluster(ctrl), nil
				}

				createResponse := &api.CreateDataplaneResponse{
					DataplaneID:  "test-dp-id",
					ClientID:     "test-client-id",
					ClientSecret: "test-secret",
				}

				gomock.InOrder(
					mockUI.EXPECT().Title("Starting interactive dataplane installation"),
					mockStore.EXPECT().Load().Return(config, nil),
					mockUI.EXPECT().Select("Select region option:", []string{"Use existing region", "Create new region"}).Return(0, "Use existing region", nil),
					mockService.EXPECT().ListRegions(gomock.Any(), "test-org-id").Return([]*api.Region{
						{ID: "region-1", Name: "Test Region", Location: "US", CloudProvider: "AWS"},
					}, nil),
					mockUI.EXPECT().FilterableSelect("Select existing region:", []string{"Test Region - US (AWS)"}).Return(0, "Test Region - US (AWS)", nil),
					mockUI.EXPECT().TextInput("Enter dataplane name:", "my-dataplane", gomock.Any()).Return("test-dataplane", nil),
					mockUI.EXPECT().RunWithSpinner("Creating dataplane", gomock.Any()).DoAndReturn(func(msg string, fn func() error) error {
						return fn()
					}),
					mockService.EXPECT().CreateDataplane(gomock.Any(), api.CreateDataplaneRequest{
						Name:           "test-dataplane",
						RegionID:       "region-1",
						OrganizationID: "test-org-id",
						Enabled:        true,
					}).Return(createResponse, nil),
					mockUI.EXPECT().ShowSection("Dataplane Credentials:",
						"DataplaneID: test-dp-id",
						"ClientID: test-client-id",
						"ClientSecret: test-secret"),
					mockUI.EXPECT().RunWithSpinner("Installing dataplane chart", gomock.Any()).DoAndReturn(func(msg string, fn func() error) error {
						return fn()
					}),
					mockHelmClient.EXPECT().AddOrUpdateChartRepo(gomock.Any()).Return(nil),
					mockHelmClient.EXPECT().InstallOrUpgradeChart(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil),
					mockUI.EXPECT().ShowSuccess("Dataplane 'test-dataplane' installed successfully!"),
					mockUI.EXPECT().NewLine(),
				)

				return mockStore, mockHTTP, mockAPIFactory, mockHelmFactory, mockClusterFactory, mockUI
			},
		},
		{
			name:      "success flow with Kind cluster",
			withKind:  true,
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, helm.Factory, k8s.ClusterFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)
				mockHelmClient := helmmock.NewMockClient(ctrl)
				mockCluster := k8smock.NewMockCluster(ctrl)

				config := &airbox.Config{
					CurrentContext: "https://test.airbyte.com",
					Contexts: []airbox.NamedContext{
						{
							Name: "https://test.airbyte.com",
							Context: airbox.Context{
								OrganizationID: "test-org-id",
							},
						},
					},
				}

				mockAPIFactory := func(ctx context.Context, httpClient http.HTTPDoer, cfg airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				mockHelmFactory := func(kubeConfig, kubeContext, namespace string) (goHelm.Client, error) {
					return mockHelmClient, nil
				}

				mockClusterFactory := func(ctx context.Context, clusterName string) (k8s.Cluster, error) {
					assert.Equal(t, "airbox-test-dataplane", clusterName)
					return mockCluster, nil
				}

				createResponse := &api.CreateDataplaneResponse{
					DataplaneID:  "test-dp-id",
					ClientID:     "test-client-id",
					ClientSecret: "test-secret",
				}

				gomock.InOrder(
					mockUI.EXPECT().Title("Starting interactive dataplane installation"),
					mockStore.EXPECT().Load().Return(config, nil),
					mockUI.EXPECT().Select("Select region option:", []string{"Use existing region", "Create new region"}).Return(0, "Use existing region", nil),
					mockService.EXPECT().ListRegions(gomock.Any(), "test-org-id").Return([]*api.Region{
						{ID: "region-1", Name: "Test Region", Location: "US", CloudProvider: "AWS"},
					}, nil),
					mockUI.EXPECT().FilterableSelect("Select existing region:", []string{"Test Region - US (AWS)"}).Return(0, "Test Region - US (AWS)", nil),
					mockUI.EXPECT().TextInput("Enter dataplane name:", "my-dataplane", gomock.Any()).Return("test-dataplane", nil),
					mockUI.EXPECT().RunWithSpinner("Creating Kind cluster", gomock.Any()).DoAndReturn(func(msg string, fn func() error) error {
						return fn()
					}),
					mockCluster.EXPECT().Create(gomock.Any(), 0, nil).Return(nil),
					mockUI.EXPECT().RunWithSpinner("Creating dataplane", gomock.Any()).DoAndReturn(func(msg string, fn func() error) error {
						return fn()
					}),
					mockService.EXPECT().CreateDataplane(gomock.Any(), api.CreateDataplaneRequest{
						Name:           "test-dataplane",
						RegionID:       "region-1",
						OrganizationID: "test-org-id",
						Enabled:        true,
					}).Return(createResponse, nil),
					mockUI.EXPECT().ShowSection("Dataplane Credentials:",
						"DataplaneID: test-dp-id",
						"ClientID: test-client-id",
						"ClientSecret: test-secret"),
					mockUI.EXPECT().RunWithSpinner("Installing dataplane chart", gomock.Any()).DoAndReturn(func(msg string, fn func() error) error {
						return fn()
					}),
					mockHelmClient.EXPECT().AddOrUpdateChartRepo(gomock.Any()).Return(nil),
					mockHelmClient.EXPECT().InstallOrUpgradeChart(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil),
					mockUI.EXPECT().ShowSuccess("Dataplane 'test-dataplane' installed successfully!"),
					mockUI.EXPECT().NewLine(),
					mockUI.EXPECT().ShowHeading("To use kubectl with this Kind cluster"),
					mockUI.EXPECT().ShowKeyValue("1. Export kubeconfig", "kind export kubeconfig --name airbox-test-dataplane"),
					mockUI.EXPECT().NewLine(),
				)

				return mockStore, mockHTTP, mockAPIFactory, mockHelmFactory, mockClusterFactory, mockUI
			},
		},
		{
			name:      "success flow with Kind cluster and custom namespace",
			withKind:  true,
			namespace: "custom",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, http.HTTPDoer, airbox.APIServiceFactory, helm.Factory, k8s.ClusterFactory, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockHTTP := httpmock.NewMockHTTPDoer(ctrl)
				mockService := apimock.NewMockService(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)
				mockHelmClient := helmmock.NewMockClient(ctrl)
				mockCluster := k8smock.NewMockCluster(ctrl)

				config := &airbox.Config{
					CurrentContext: "https://test.airbyte.com",
					Contexts: []airbox.NamedContext{
						{
							Name: "https://test.airbyte.com",
							Context: airbox.Context{
								OrganizationID: "test-org-id",
							},
						},
					},
				}

				mockAPIFactory := func(ctx context.Context, httpClient http.HTTPDoer, cfg airbox.ConfigStore) (api.Service, error) {
					return mockService, nil
				}

				mockHelmFactory := func(kubeConfig, kubeContext, namespace string) (goHelm.Client, error) {
					return mockHelmClient, nil
				}

				mockClusterFactory := func(ctx context.Context, clusterName string) (k8s.Cluster, error) {
					assert.Equal(t, "airbox-test-dataplane", clusterName)
					return mockCluster, nil
				}

				createResponse := &api.CreateDataplaneResponse{
					DataplaneID:  "test-dp-id",
					ClientID:     "test-client-id",
					ClientSecret: "test-secret",
				}

				gomock.InOrder(
					mockUI.EXPECT().Title("Starting interactive dataplane installation"),
					mockStore.EXPECT().Load().Return(config, nil),
					mockUI.EXPECT().Select("Select region option:", []string{"Use existing region", "Create new region"}).Return(0, "Use existing region", nil),
					mockService.EXPECT().ListRegions(gomock.Any(), "test-org-id").Return([]*api.Region{
						{ID: "region-1", Name: "Test Region", Location: "US", CloudProvider: "AWS"},
					}, nil),
					mockUI.EXPECT().FilterableSelect("Select existing region:", []string{"Test Region - US (AWS)"}).Return(0, "Test Region - US (AWS)", nil),
					mockUI.EXPECT().TextInput("Enter dataplane name:", "my-dataplane", gomock.Any()).Return("test-dataplane", nil),
					mockUI.EXPECT().RunWithSpinner("Creating Kind cluster", gomock.Any()).DoAndReturn(func(msg string, fn func() error) error {
						return fn()
					}),
					mockCluster.EXPECT().Create(gomock.Any(), 0, nil).Return(nil),
					mockUI.EXPECT().RunWithSpinner("Creating dataplane", gomock.Any()).DoAndReturn(func(msg string, fn func() error) error {
						return fn()
					}),
					mockService.EXPECT().CreateDataplane(gomock.Any(), api.CreateDataplaneRequest{
						Name:           "test-dataplane",
						RegionID:       "region-1",
						OrganizationID: "test-org-id",
						Enabled:        true,
					}).Return(createResponse, nil),
					mockUI.EXPECT().ShowSection("Dataplane Credentials:",
						"DataplaneID: test-dp-id",
						"ClientID: test-client-id",
						"ClientSecret: test-secret"),
					mockUI.EXPECT().RunWithSpinner("Installing dataplane chart", gomock.Any()).DoAndReturn(func(msg string, fn func() error) error {
						return fn()
					}),
					mockHelmClient.EXPECT().AddOrUpdateChartRepo(gomock.Any()).Return(nil),
					mockHelmClient.EXPECT().InstallOrUpgradeChart(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil),
					mockUI.EXPECT().ShowSuccess("Dataplane 'test-dataplane' installed successfully!"),
					mockUI.EXPECT().NewLine(),
					mockUI.EXPECT().ShowHeading("To use kubectl with this Kind cluster"),
					mockUI.EXPECT().ShowKeyValue("1. Export kubeconfig", "kind export kubeconfig --name airbox-test-dataplane"),
					mockUI.EXPECT().ShowKeyValue("2. Set namespace", "kubectl config set-context --current --namespace=custom"),
					mockUI.EXPECT().NewLine(),
				)

				return mockStore, mockHTTP, mockAPIFactory, mockHelmFactory, mockClusterFactory, mockUI
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cmd := &DataplaneCmd{
				Namespace:       tt.namespace,
				WithKindCluster: tt.withKind,
			}

			cfg, httpClient, apiFactory, helmFactory, clusterFactory, ui := tt.setupMocks(ctrl)

			err := cmd.Run(context.Background(), cfg, httpClient, apiFactory, helmFactory, clusterFactory, ui)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

