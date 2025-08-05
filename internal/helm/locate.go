package helm

import (
	"errors"
	"fmt"
	"strings"

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
