package helm

import (
	"net/http"

	goHelm "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// GetMetadataForRef returns the chart metadata for a local path or chart reference using the provided Helm client.
func GetMetadataForRef(client goHelm.Client, chartRef string) (*chart.Metadata, error) {
	chart, _, err := client.GetChart(chartRef, &action.ChartPathOptions{})
	if err != nil {
		return nil, err
	}

	return chart.Metadata, nil
}

// GetMetadataForURL fetches a remote chart archive (.tgz) from the given URL and returns its chart metadata.
func GetMetadataForURL(url string) (*chart.Metadata, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	chart, err := loader.LoadArchive(resp.Body)
	if err != nil {
		return nil, err
	}

	return chart.Metadata, nil
}
