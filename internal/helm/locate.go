package helm

import (
	"errors"
	"fmt"
	"strings"

	"github.com/airbytehq/abctl/internal/common"
	"github.com/pterm/pterm"
	"golang.org/x/mod/semver"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

// chartRepo exists only for testing purposes.
// This allows the DownloadIndexFile method to be mocked.
type chartRepo interface {
	DownloadIndexFile() (string, error)
}

var _ chartRepo = (*repo.ChartRepository)(nil)

// newChartRepo exists only for testing purposes.
// This allows a test implementation of the repo.NewChartRepository function to exist.
type newChartRepo func(cfg *repo.Entry, getters getter.Providers) (chartRepo, error)

// loadIndexFile exists only for testing purposes.
// This allows a test implementation of the repo.LoadIndexFile function to exist.
type loadIndexFile func(path string) (*repo.IndexFile, error)

// defaultNewChartRepo is the default implementation of the newChartRepo function.
// It simply wraps the repo.NewChartRepository function.
// This variable should only be modified for testing purposes.
var defaultNewChartRepo newChartRepo = func(cfg *repo.Entry, getters getter.Providers) (chartRepo, error) {
	return repo.NewChartRepository(cfg, getters)
}

// defaultLoadIndexFile is the default implementation of the loadIndexFile function.
// It simply wraps the repo.LoadIndexFile function.
// This variable should only be modified for testing purposes.
var defaultLoadIndexFile loadIndexFile = repo.LoadIndexFile

func LocateLatestAirbyteChart(chartVersion, chartFlag string) string {
	pterm.Debug.Printf("getting helm chart %q with version %q\n", common.AirbyteChartName, chartVersion)

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
	if chartVersion == "" {
		if url, _, err := GetLatestAirbyteChartUrlFromRepoIndex(common.AirbyteRepoName, common.AirbyteRepoURLv1); err == nil {
			pterm.Debug.Printf("determined latest airbyte chart url: %s\n", url)
			return url
		} else {
			pterm.Debug.Printf("error determining latest airbyte chart, falling back to default behavior: %s\n", err)
		}
	}

	return common.AirbyteChartName
}

// GetLatestAirbyteChartUrlFromRepoIndex fetches the latest stable Airbyte Helm chart URL and version
// from the given Helm repository index. Returns the chart download URL, the chart version, and an error if any.
// Only stable (non-prerelease) versions are considered.
func GetLatestAirbyteChartUrlFromRepoIndex(repoName, repoUrl string) (string, string, error) {
	chartRepository, err := defaultNewChartRepo(&repo.Entry{
		Name: repoName,
		URL:  repoUrl,
	}, getter.All(cli.New()))
	if err != nil {
		return "", "", fmt.Errorf("unable to access repo index: %w", err)
	}

	idxPath, err := chartRepository.DownloadIndexFile()
	if err != nil {
		return "", "", fmt.Errorf("unable to download index file: %w", err)
	}

	idx, err := defaultLoadIndexFile(idxPath)
	if err != nil {
		return "", "", fmt.Errorf("unable to load index file (%s): %w", idxPath, err)
	}

	entries, ok := idx.Entries["airbyte"]
	if !ok {
		return "", "", fmt.Errorf("no entry for airbyte in repo index")
	}

	if len(entries) == 0 {
		return "", "", errors.New("no chart version found")
	}

	var latest *repo.ChartVersion
	for _, entry := range entries {
		version := entry.Version
		// the semver library requires a `v` prefix
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}

		if semver.Prerelease(version) == "" {
			latest = entry
			break
		}
	}

	if latest == nil {
		return "", "", fmt.Errorf("no valid version of airbyte chart found in repo index")
	}

	if len(latest.URLs) != 1 {
		return "", "", fmt.Errorf("unexpected number of URLs - %d", len(latest.URLs))
	}

	return repoUrl + "/" + latest.URLs[0], latest.Version, nil
}
