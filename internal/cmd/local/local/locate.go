package local

import (
	"fmt"

	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

func locateLatestAirbyteChart(chartName, chartVersion, chartFlag string) string {
	pterm.Debug.Printf("getting helm chart %q with version %q\n", chartName, chartVersion)

	// If the --chart flag was given, use that.
	if chartFlag != "" {
		return chartFlag
	}

	// Helm will consider a local directory path named "airbyte/airbyte" to be a chart repo,
	// but it might not be, which causes errors like "Chart.yaml file is missing".
	// This trips up plenty of people, see: https://github.com/helm/helm/issues/7862
	//
	// Here we avoid that problem by figuring out the full URL of the airbyte chart,
	// which forces Helm to resolve the chart over HTTP and ignore local directories.
	// If the locator fails, fall back to the original helm behavior.
	if chartName == airbyteChartName && chartVersion == "" {
		if url, err := getLatestAirbyteChartUrlFromRepoIndex(airbyteRepoName, airbyteRepoURL); err == nil {
			pterm.Debug.Printf("determined latest airbyte chart url: %s\n", url)
			return url
		} else {
			pterm.Debug.Printf("error determining latest airbyte chart, falling back to default behavior: %s\n", err)
		}
	}

	return chartName
}

func getLatestAirbyteChartUrlFromRepoIndex(repoName, repoUrl string) (string, error) {
	chartRepo, err := repo.NewChartRepository(&repo.Entry{
		Name: repoName,
		URL:  repoUrl,
	}, getter.All(cli.New()))
	if err != nil {
		return "", fmt.Errorf("unable to access repo index: %w", err)
	}

	idxPath, err := chartRepo.DownloadIndexFile()
	if err != nil {
		return "", fmt.Errorf("unable to download index file: %w", err)
	}

	idx, err := repo.LoadIndexFile(idxPath)
	if err != nil {
		return "", fmt.Errorf("unable to load index file (%s): %w", idxPath, err)
	}

	airbyteEntry, ok := idx.Entries["airbyte"]
	if !ok {
		return "", fmt.Errorf("no entry for airbyte in repo index")
	}

	if len(airbyteEntry) == 0 {
		return "", fmt.Errorf("no chart version found")
	}

	latest := airbyteEntry[0]
	if len(latest.URLs) != 1 {
		return "", fmt.Errorf("unexpected number of URLs")
	}
	return airbyteRepoURL + "/" + latest.URLs[0], nil
}
