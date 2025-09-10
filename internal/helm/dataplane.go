package helm

import (
	"context"
	"fmt"
	"time"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/api"
	"github.com/airbytehq/abctl/internal/common"
	goHelm "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/repo"
)

const (
	dataplaneChartName    = "airbyte/airbyte-data-plane"
	dataplaneChartVersion = "1.8.1"
	dataplaneRepoName     = "airbyte"
	dataplaneRepoURL      = common.AirbyteRepoURLv1 // v1 repo has the dataplane chart
)

// InstallDataplaneChart installs the official Airbyte dataplane Helm chart
func InstallDataplaneChart(ctx context.Context, client goHelm.Client, namespace, releaseName string, credentials *api.CreateDataplaneResponse, config *airbox.Context) error {
	// Add the Airbyte Helm repository
	if err := client.AddOrUpdateChartRepo(repo.Entry{
		Name: dataplaneRepoName,
		URL:  dataplaneRepoURL,
	}); err != nil {
		return fmt.Errorf("failed to add Airbyte chart repository: %w", err)
	}

	// Build values for the dataplane chart
	valuesYAML := buildDataplaneValues(credentials, config)

	// Install the chart with atomic flag to ensure cleanup on failure/interrupt
	_, err := client.InstallOrUpgradeChart(ctx, &goHelm.ChartSpec{
		ReleaseName:     releaseName,
		ChartName:       dataplaneChartName,
		Version:         dataplaneChartVersion,
		CreateNamespace: true,
		Namespace:       namespace,
		Wait:            true,
		Atomic:          true, // Rollback on failure, including Ctrl-C
		Timeout:         10 * time.Minute,
		ValuesYaml:      valuesYAML,
	}, &goHelm.GenericHelmOptions{})
	if err != nil {
		return fmt.Errorf("failed to install dataplane chart: %w", err)
	}

	return nil
}

// buildDataplaneValues constructs the Helm values YAML for the dataplane chart
func buildDataplaneValues(credentials *api.CreateDataplaneResponse, config *airbox.Context) string {
	return fmt.Sprintf(`# Airbyte Dataplane Configuration
# Set the Airbyte Control Plane URL
airbyteUrl: "%s"

# Workload API server configuration for remote dataplane
workloadApiServer:
  url: "%s"

dataPlane:
  id: "%s"
  clientId: "%s"
  clientSecret: "%s"

# Configure for dataplane mode
edition: "%s"

# Storage configuration (using default Minio for now)
storage:
  type: minio
  minio:
    accessKeyId: minio
    secretAccessKey: "minio123"
  bucket:
    log: airbyte-bucket
    state: airbyte-bucket
    workloadOutput: airbyte-bucket
    activityPayload: airbyte-bucket

# Workloads configuration
workloads:
  namespace: "jobs"

# Logging configuration
logging:
  level: info

# Metrics disabled by default
metrics:
  enabled: "false"
`, config.AirbyteURL, config.AirbyteAPIURL, credentials.DataplaneID, credentials.ClientID, credentials.ClientSecret, config.Edition)
}
