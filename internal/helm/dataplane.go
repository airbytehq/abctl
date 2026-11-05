package helm

import (
	"context"
	"fmt"
	"time"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/api"
	"github.com/airbytehq/abctl/internal/common"
	goHelm "github.com/mittwald/go-helm-client"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/repo"
)

const (
	dataplaneRepoName  = "airbyte"
	dataplaneChartName = "airbyte-data-plane"
)

var getLatestDataplaneChartURL = func(repoURL string) (string, string, error) {
	return GetLatestChartURLFromRepoIndex(dataplaneRepoName, repoURL, dataplaneChartName)
}

// ResolveDataplaneChartReference resolves the dataplane chart URL, version and
// source repository for the requested version. An empty version resolves the
// latest stable chart from the v2 repository; a version below 2.0.0 resolves
// against the deprecated v1 repository.
func ResolveDataplaneChartReference(version string) (chartURL, chartVersion, repoURL string, err error) {
	if version == "" {
		chartURL, chartVersion, err = getLatestDataplaneChartURL(common.AirbyteRepoURLv2)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to resolve latest dataplane chart: %w", err)
		}
		return chartURL, chartVersion, common.AirbyteRepoURLv2, nil
	}

	repoURL = common.AirbyteRepoURLv1
	if ChartIsV2PlusBaseVersion(version) {
		repoURL = common.AirbyteRepoURLv2
	}

	return fmt.Sprintf("%s/%s-%s.tgz", repoURL, dataplaneChartName, version), version, repoURL, nil
}

type DataplaneChartOptions struct {
	Values        map[string]any
	TargetCluster string
	DeferWait     bool
}

// InstallDataplaneChart installs the official Airbyte dataplane Helm chart.
func InstallDataplaneChart(ctx context.Context, client goHelm.Client, namespace, releaseName, chartURL, chartVersion, repoURL string, credentials *api.CreateDataplaneResponse, config *airbox.Context, options ...DataplaneChartOptions) error {
	// Add the Airbyte Helm repository
	if err := client.AddOrUpdateChartRepo(repo.Entry{
		Name: dataplaneRepoName,
		URL:  repoURL,
	}); err != nil {
		return fmt.Errorf("failed to add Airbyte chart repository: %w", err)
	}

	// Build values for the dataplane chart
	valuesYAML := buildDataplaneValues(credentials, config)
	wait := true
	if len(options) > 0 {
		var base map[string]any
		if err := yaml.Unmarshal([]byte(valuesYAML), &base); err != nil {
			return fmt.Errorf("dataplane defaults are not valid YAML")
		}
		merged := MergeValues(base, options[0].Values)
		// Installation identity and routing cannot be overridden by an overlay.
		merged["dataPlane"] = base["dataPlane"]
		merged["targetCluster"] = options[0].TargetCluster
		data, err := yaml.Marshal(merged)
		if err != nil {
			return fmt.Errorf("dataplane values cannot be encoded")
		}
		valuesYAML = string(data)
		wait = !options[0].DeferWait
	}

	_, err := client.InstallOrUpgradeChart(ctx, &goHelm.ChartSpec{
		ReleaseName:     releaseName,
		ChartName:       chartURL,
		Version:         chartVersion,
		CreateNamespace: true,
		Namespace:       namespace,
		Wait:            wait,
		Atomic:          wait,
		Timeout:         10 * time.Minute,
		ValuesYaml:      valuesYAML,
	}, &goHelm.GenericHelmOptions{})
	if err != nil {
		return fmt.Errorf("failed to install dataplane chart: %w", err)
	}

	return nil
}

func MergeValues(base, overlay map[string]any) map[string]any {
	result := make(map[string]any, len(base)+len(overlay))
	for key, value := range base {
		result[key] = value
	}
	for key, value := range overlay {
		left, leftOK := result[key].(map[string]any)
		right, rightOK := value.(map[string]any)
		if leftOK && rightOK {
			result[key] = MergeValues(left, right)
		} else {
			result[key] = value
		}
	}
	return result
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
