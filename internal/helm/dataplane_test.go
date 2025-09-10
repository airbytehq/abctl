package helm

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/api"
	"github.com/airbytehq/abctl/internal/helm/mock"
	goHelm "github.com/mittwald/go-helm-client"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
)

func TestBuildDataplaneValues(t *testing.T) {
	tests := []struct {
		name        string
		credentials *api.CreateDataplaneResponse
		config      *airbox.Context
		wantStrings []string
		notWant     []string
	}{
		{
			name: "enterprise configuration",
			credentials: &api.CreateDataplaneResponse{
				DataplaneID:  "test-dataplane-id",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			},
			config: &airbox.Context{
				AirbyteURL:    "https://enterprise.example.com",
				AirbyteAPIURL: "https://enterprise.example.com/api/v1",
				Edition:       "enterprise",
			},
			wantStrings: []string{
				`airbyteUrl: "https://enterprise.example.com"`,
				`url: "https://enterprise.example.com/api/v1"`,
				`id: "test-dataplane-id"`,
				`clientId: "test-client-id"`,
				`clientSecret: "test-client-secret"`,
				`edition: "enterprise"`,
				`type: minio`,
				`namespace: "jobs"`,
			},
			notWant: []string{
				"extraEnv",
				"INTERNAL_API_HOST",
			},
		},
		{
			name: "cloud configuration",
			credentials: &api.CreateDataplaneResponse{
				DataplaneID:  "cloud-dataplane-id",
				ClientID:     "cloud-client-id",
				ClientSecret: "cloud-client-secret",
			},
			config: &airbox.Context{
				AirbyteURL:    "https://cloud.airbyte.com",
				AirbyteAPIURL: "https://api.airbyte.com/v1",
				Edition:       "cloud",
			},
			wantStrings: []string{
				`airbyteUrl: "https://cloud.airbyte.com"`,
				`url: "https://api.airbyte.com/v1"`,
				`id: "cloud-dataplane-id"`,
				`clientId: "cloud-client-id"`,
				`clientSecret: "cloud-client-secret"`,
				`edition: "cloud"`,
			},
		},
		{
			name: "community edition",
			credentials: &api.CreateDataplaneResponse{
				DataplaneID:  "community-dataplane-id",
				ClientID:     "community-client-id",
				ClientSecret: "community-client-secret",
			},
			config: &airbox.Context{
				AirbyteURL:    "http://localhost:8000",
				AirbyteAPIURL: "http://localhost:8000/api/v1",
				Edition:       "community",
			},
			wantStrings: []string{
				`airbyteUrl: "http://localhost:8000"`,
				`url: "http://localhost:8000/api/v1"`,
				`edition: "community"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildDataplaneValues(tt.credentials, tt.config)

			// Check that expected strings are present
			for _, want := range tt.wantStrings {
				if !strings.Contains(got, want) {
					t.Errorf("buildDataplaneValues() missing expected string: %q\nGot:\n%s", want, got)
				}
			}

			// Check that unwanted strings are not present
			for _, notWant := range tt.notWant {
				if strings.Contains(got, notWant) {
					t.Errorf("buildDataplaneValues() contains unwanted string: %q\nGot:\n%s", notWant, got)
				}
			}
		})
	}
}

func TestInstallDataplaneChart(t *testing.T) {
	tests := []struct {
		name              string
		namespace         string
		releaseName       string
		credentials       *api.CreateDataplaneResponse
		config            *airbox.Context
		repoError         error
		installError      error
		expectedRepoEntry repo.Entry
		expectedError     string
	}{
		{
			name:        "successful installation",
			namespace:   "test-namespace",
			releaseName: "test-dataplane",
			credentials: &api.CreateDataplaneResponse{
				DataplaneID:  "test-dataplane-id",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			},
			config: &airbox.Context{
				AirbyteURL:    "https://test.example.com",
				AirbyteAPIURL: "https://test.example.com/api/v1",
				Edition:       "enterprise",
			},
			expectedRepoEntry: repo.Entry{
				Name: dataplaneRepoName,
				URL:  dataplaneRepoURL,
			},
		},
		{
			name:        "repository addition fails",
			namespace:   "test-namespace",
			releaseName: "test-dataplane",
			credentials: &api.CreateDataplaneResponse{
				DataplaneID:  "test-dataplane-id",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			},
			config: &airbox.Context{
				AirbyteURL:    "https://test.example.com",
				AirbyteAPIURL: "https://test.example.com/api/v1",
				Edition:       "enterprise",
			},
			expectedRepoEntry: repo.Entry{
				Name: dataplaneRepoName,
				URL:  dataplaneRepoURL,
			},
			repoError:     assert.AnError,
			expectedError: "failed to add Airbyte chart repository",
		},
		{
			name:        "chart installation fails",
			namespace:   "test-namespace",
			releaseName: "test-dataplane",
			credentials: &api.CreateDataplaneResponse{
				DataplaneID:  "test-dataplane-id",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			},
			config: &airbox.Context{
				AirbyteURL:    "https://test.example.com",
				AirbyteAPIURL: "https://test.example.com/api/v1",
				Edition:       "enterprise",
			},
			expectedRepoEntry: repo.Entry{
				Name: dataplaneRepoName,
				URL:  dataplaneRepoURL,
			},
			installError:  assert.AnError,
			expectedError: "failed to install dataplane chart",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mock.NewMockClient(ctrl)

			// Set up expectations for AddOrUpdateChartRepo
			mockClient.EXPECT().AddOrUpdateChartRepo(tt.expectedRepoEntry).Return(tt.repoError)

			if tt.repoError == nil {
				// Set up expectations for InstallOrUpgradeChart
				mockClient.EXPECT().InstallOrUpgradeChart(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, spec *goHelm.ChartSpec, opts *goHelm.GenericHelmOptions) (*release.Release, error) {
						// Verify chart spec parameters
						assert.Equal(t, tt.releaseName, spec.ReleaseName)
						assert.Equal(t, dataplaneChartName, spec.ChartName)
						assert.Equal(t, dataplaneChartVersion, spec.Version)
						assert.Equal(t, tt.namespace, spec.Namespace)
						assert.True(t, spec.CreateNamespace)
						assert.True(t, spec.Wait)
						assert.True(t, spec.Atomic)
						assert.Equal(t, 10*time.Minute, spec.Timeout)

						// Verify the values contain expected dataplane configuration
						assert.Contains(t, spec.ValuesYaml, tt.credentials.DataplaneID)
						assert.Contains(t, spec.ValuesYaml, tt.credentials.ClientID)
						assert.Contains(t, spec.ValuesYaml, tt.credentials.ClientSecret)
						assert.Contains(t, spec.ValuesYaml, tt.config.AirbyteURL)

						return nil, tt.installError
					})
			}

			err := InstallDataplaneChart(context.Background(), mockClient, tt.namespace, tt.releaseName, tt.credentials, tt.config)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
