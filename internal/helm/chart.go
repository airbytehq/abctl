package helm

import (
	"fmt"

	"github.com/airbytehq/abctl/internal/common"
	"github.com/airbytehq/abctl/internal/validate"
	goHelm "github.com/mittwald/go-helm-client"
	"golang.org/x/mod/semver"
)

// ChartResolver handles resolution of Airbyte chart references from v1/v2 repositories, local paths, and URLs
type ChartResolver struct {
	// v1RepoURL is the base URL for v1 Airbyte helm charts
	v1RepoURL string
	// v2RepoURL is the base URL for v2 Airbyte helm charts
	v2RepoURL string
	// client provides Chart.yaml metadata extraction for local paths and URLs
	client goHelm.Client
}

// NewChartResolver creates a resolver with default Airbyte v1/v2 repository URLs
func NewChartResolver(client goHelm.Client) *ChartResolver {
	return &ChartResolver{
		v1RepoURL: common.AirbyteRepoURLv1,
		v2RepoURL: common.AirbyteRepoURLv2,
		client:    client,
	}
}

// NewChartResolverWithURLs creates a resolver with custom v1/v2 repository URLs
func NewChartResolverWithURLs(client goHelm.Client, v1URL, v2URL string) *ChartResolver {
	return &ChartResolver{
		v1RepoURL: v1URL,
		v2RepoURL: v2URL,
		client:    client,
	}
}

// ChartIsV2Plus returns true if the chart version is v2.0.0 or higher
func ChartIsV2Plus(v string) bool {
	if v == "" {
		return false
	}
	if v[0] != 'v' {
		v = "v" + v
	}
	return semver.Compare(v, "v2.0.0") >= 0
}

// ResolveChartReference resolves an Airbyte chart reference to its full URL/path and version.
// For empty chart+version, returns latest v2 chart.
// For version-only, uses v1/v2 repo based on version number.
// For URLs and local paths, returns as-is with version from chart metadata.
func (r *ChartResolver) ResolveChartReference(chart, version string) (string, string, error) {
	if chart == "" {
		if version == "" {
			chartURL, chartVersion, err := GetLatestAirbyteChartUrlFromRepoIndex("", r.v2RepoURL)
			if err != nil {
				return "", "", err
			}
			return chartURL, chartVersion, nil
		} else {
			if ChartIsV2Plus(version) {
				// Construct the v2 chart URL.
				return fmt.Sprintf("%s/airbyte-%s.tgz", r.v2RepoURL, version), version, nil
			} else {
				// Construct the v1 chart URL.
				return fmt.Sprintf("%s/airbyte-%s.tgz", r.v1RepoURL, version), version, nil
			}
		}
	}

	// Since chart and version are mutually exclusive flags. Get the latest version for the chart.
	if validate.IsURL(chart) {
		meta, err := GetMetadataForURL(chart)
		if err != nil {
			return "", "", fmt.Errorf("failed to get helm chart metadata for url %s: %w", chart, err)
		}
		return chart, meta.Version, nil
	}

	meta, err := GetMetadataForRef(r.client, chart)
	if err != nil {
		return "", "", fmt.Errorf("failed to get helm chart metadata for reference %s: %w", chart, err)
	}

	return chart, meta.Version, nil
}
