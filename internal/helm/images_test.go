package helm

import (
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"helm.sh/helm/v3/pkg/release"

	"github.com/airbytehq/abctl/internal/helm/mock"
)

func TestFindImagesFromChart(t *testing.T) {
	testCases := []struct {
		name             string
		valuesPath       string
		chartName        string
		chartVersion     string
		expect           []string
		mockSetup        func(client *mock.MockClient)
		expectNilNoError bool
	}{
		{
			name:         "enterprise-values-v1.1.0",
			valuesPath:   "../cmd/images/testdata/enterprise.values.yaml",
			chartName:    "airbyte/airbyte",
			chartVersion: "1.1.0",
			expect: []string{
				"airbyte/bootloader:1.1.0",
				"airbyte/connector-builder-server:1.1.0",
				"airbyte/cron:1.1.0",
				"airbyte/db:1.1.0",
				"airbyte/keycloak-setup:1.1.0",
				"airbyte/keycloak:1.1.0",
				"airbyte/mc",
				"airbyte/server:1.1.0",
				"airbyte/webapp:1.1.0",
				"airbyte/worker:1.1.0",
				"airbyte/workload-api-server:1.1.0",
				"airbyte/workload-launcher:1.1.0",
				"bitnami/kubectl:1.28.9",
				"busybox",
				"curlimages/curl:8.1.1",
				"minio/minio:RELEASE.2023-11-20T22-40-07Z",
				"postgres:13-alpine",
				"temporalio/auto-setup:1.23.0",
			},
			mockSetup: func(client *mock.MockClient) {
				client.EXPECT().AddOrUpdateChartRepo(gomock.Any()).Return(nil)
				client.EXPECT().InstallChart(gomock.Any(), gomock.Any(), gomock.Any()).Return(&release.Release{Manifest: sampleRenderedYaml}, nil)
			},
		},
		{
			name:         "configmap-airbyte-env-image-keys",
			valuesPath:   "",
			chartName:    "airbyte/airbyte",
			chartVersion: "1.1.0",
			expect:       []string{"img1:v1", "img2:v2", "shouldnotinclude"},
			mockSetup: func(client *mock.MockClient) {
				client.EXPECT().
					AddOrUpdateChartRepo(gomock.Any()).
					Return(nil)
				client.EXPECT().
					InstallChart(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&release.Release{Manifest: configMapRenderedYaml}, nil)
			},
		},
		{
			name:         "all-kinds-containers-initcontainers",
			valuesPath:   "",
			chartName:    "airbyte/airbyte",
			chartVersion: "1.1.0",
			expect:       []string{"podimg", "jobimg", "deployimg", "statefulimg", "initimg"},
			mockSetup: func(client *mock.MockClient) {
				client.EXPECT().
					AddOrUpdateChartRepo(gomock.Any()).
					Return(nil)
				client.EXPECT().
					InstallChart(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&release.Release{Manifest: allKindsRenderedYaml}, nil)
			},
		},
		{
			name:         "addorupdatechartrepo-error",
			valuesPath:   "",
			chartName:    "airbyte/airbyte",
			chartVersion: "1.1.0",
			expect:       nil,
			mockSetup: func(client *mock.MockClient) {
				client.EXPECT().
					AddOrUpdateChartRepo(gomock.Any()).
					Return(assert.AnError)
			},
		},
		{
			name:         "installchart-error",
			valuesPath:   "",
			chartName:    "airbyte/airbyte",
			chartVersion: "1.1.0",
			expect:       nil,
			mockSetup: func(client *mock.MockClient) {
				client.EXPECT().
					AddOrUpdateChartRepo(gomock.Any()).
					Return(nil)
				client.EXPECT().
					InstallChart(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)
			},
		},
		{
			name:         "decodek8sresources-empty-invalid-chunks",
			valuesPath:   "",
			chartName:    "airbyte/airbyte",
			chartVersion: "1.1.0",
			expect:       []string{"podimg"},
			mockSetup: func(client *mock.MockClient) {
				client.EXPECT().
					AddOrUpdateChartRepo(gomock.Any()).
					Return(nil)
				client.EXPECT().
					InstallChart(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&release.Release{Manifest: decodeContinuesRenderedYaml}, nil)
			},
		},
		{
			name:         "decodek8sresources-empty-chunk-only",
			valuesPath:   "",
			chartName:    "airbyte/airbyte",
			chartVersion: "1.1.0",
			expect:       nil,
			mockSetup: func(client *mock.MockClient) {
				client.EXPECT().
					AddOrUpdateChartRepo(gomock.Any()).
					Return(nil)
				client.EXPECT().
					InstallChart(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&release.Release{Manifest: "---\n"}, nil)
			},
			expectNilNoError: true,
		},
		{
			name:         "decodek8sresources-unknown-kind-ignored",
			valuesPath:   "",
			chartName:    "airbyte/airbyte",
			chartVersion: "1.1.0",
			expect:       nil,
			mockSetup: func(client *mock.MockClient) {
				client.EXPECT().
					AddOrUpdateChartRepo(gomock.Any()).
					Return(nil)
				client.EXPECT().
					InstallChart(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&release.Release{Manifest: unknownKindRenderedYaml}, nil)
			},
			expectNilNoError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			client := mock.NewMockClient(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(client)
			}

			var valuesYaml []byte
			var err error
			if tc.valuesPath != "" {
				valuesYaml, err = os.ReadFile(tc.valuesPath)
				if err != nil {
					t.Fatalf("failed to read values yaml: %v", err)
				}
			}

			images, err := FindImagesFromChart(client, string(valuesYaml), tc.chartName, tc.chartVersion)
			if tc.expect == nil {
				if tc.expectNilNoError {
					assert.NoError(t, err)
					assert.Nil(t, images)
				} else {
					assert.Error(t, err)
				}
				return
			}
			assert.NoError(t, err)
			sort.Strings(tc.expect)
			sort.Strings(images)
			assert.Equal(t, tc.expect, images)
		})
	}
}

const configMapRenderedYaml = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-airbyte-env
  namespace: default
data:
  FOO_IMAGE: img1:v1
  BAR_IMAGE: img2:v2
  NOT_IMAGE: shouldnotinclude
`

const allKindsRenderedYaml = `
apiVersion: v1
kind: Pod
metadata:
  name: pod1
spec:
  initContainers:
    - name: init
      image: initimg
  containers:
    - name: main
      image: podimg
---
apiVersion: batch/v1
kind: Job
metadata:
  name: job1
spec:
  template:
    spec:
      initContainers:
        - name: init
          image: initimg
      containers:
        - name: main
          image: jobimg
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy1
spec:
  template:
    spec:
      initContainers:
        - name: init
          image: initimg
      containers:
        - name: main
          image: deployimg
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: stateful1
spec:
  template:
    spec:
      initContainers:
        - name: init
          image: initimg
      containers:
        - name: main
          image: statefulimg
`

const decodeContinuesRenderedYaml = `
---
# empty chunk above
invalid: [this is not valid yaml
---
apiVersion: v1
kind: Pod
metadata:
  name: pod1
spec:
  containers:
    - name: main
      image: podimg
`

const unknownKindRenderedYaml = `
apiVersion: v1
kind: Service
metadata:
  name: svc1
spec:
  selector:
    app: foo
  ports:
    - protocol: TCP
      port: 80
      targetPort: 9376
`

// sampleRenderedYaml should be a string containing a rendered Helm chart YAML with all the images above.
// For brevity, you can use a minimal YAML or mock as needed for your actual test.
const sampleRenderedYaml = `
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  containers:
    - name: bootloader
      image: airbyte/bootloader:1.1.0
    - name: connector-builder-server
      image: airbyte/connector-builder-server:1.1.0
    - name: cron
      image: airbyte/cron:1.1.0
    - name: db
      image: airbyte/db:1.1.0
    - name: keycloak-setup
      image: airbyte/keycloak-setup:1.1.0
    - name: keycloak
      image: airbyte/keycloak:1.1.0
    - name: mc
      image: airbyte/mc
    - name: server
      image: airbyte/server:1.1.0
    - name: webapp
      image: airbyte/webapp:1.1.0
    - name: worker
      image: airbyte/worker:1.1.0
    - name: workload-api-server
      image: airbyte/workload-api-server:1.1.0
    - name: workload-launcher
      image: airbyte/workload-launcher:1.1.0
    - name: kubectl
      image: bitnami/kubectl:1.28.9
    - name: busybox
      image: busybox
    - name: curl
      image: curlimages/curl:8.1.1
    - name: minio
      image: minio/minio:RELEASE.2023-11-20T22-40-07Z
    - name: postgres
      image: postgres:13-alpine
    - name: temporalio
      image: temporalio/auto-setup:1.23.0
`
