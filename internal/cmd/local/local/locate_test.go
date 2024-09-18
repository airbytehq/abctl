package local

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

func TestLocate(t *testing.T) {
	origNewChartRepo := defaultNewChartRepo
	origLoadIndexFile := defaultLoadIndexFile
	t.Cleanup(func() {
		defaultNewChartRepo = origNewChartRepo
		defaultLoadIndexFile = origLoadIndexFile
	})

	defaultNewChartRepo = mockNewChartRepo

	tests := []struct {
		name    string
		entries map[string]repo.ChartVersions
		exp     string
	}{
		{
			name: "one release entry",
			entries: map[string]repo.ChartVersions{
				"airbyte": []*repo.ChartVersion{{
					Metadata: &chart.Metadata{Version: "1.2.3"},
					URLs:     []string{"example.test"},
				}},
			},
			exp: airbyteRepoURL + "/example.test",
		},
		{
			name: "one non-release entry",
			entries: map[string]repo.ChartVersions{
				"airbyte": []*repo.ChartVersion{{
					Metadata: &chart.Metadata{Version: "1.2.3-alpha-df72e2940ca"},
					URLs:     []string{"example.test"},
				}},
			},
			exp: airbyteChartName,
		},
		{
			name:    "no entries",
			entries: map[string]repo.ChartVersions{},
			exp:     airbyteChartName,
		},
		{
			name: "one release entry with no URLs",
			entries: map[string]repo.ChartVersions{
				"airbyte": []*repo.ChartVersion{{
					Metadata: &chart.Metadata{Version: "1.2.3"},
					URLs:     []string{},
				}},
			},
			exp: airbyteChartName,
		},
		{
			name: "one release entry with two URLs",
			entries: map[string]repo.ChartVersions{
				"airbyte": []*repo.ChartVersion{{
					Metadata: &chart.Metadata{Version: "1.2.3"},
					URLs:     []string{"one.test", "two.test"},
				}},
			},
			exp: airbyteChartName,
		},
		{
			name: "one non-release entry followed by one release entry",
			entries: map[string]repo.ChartVersions{
				"airbyte": []*repo.ChartVersion{
					{
						Metadata: &chart.Metadata{Version: "1.2.3-test"},
						URLs:     []string{"bad.test"},
					},
					{
						Metadata: &chart.Metadata{Version: "0.9.8"},
						URLs:     []string{"good.test"},
					},
				},
			},
			exp: airbyteRepoURL + "/good.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultLoadIndexFile = mockLoadIndexFile(repo.IndexFile{Entries: tt.entries})
			act := locateLatestAirbyteChart(airbyteChartName, "")
			if d := cmp.Diff(tt.exp, act); d != "" {
				t.Errorf("mismatch (-want +got):\n%s", d)
			}
		})
	}
}

func mockNewChartRepo(cfg *repo.Entry, getters getter.Providers) (chartRepo, error) {
	return mockChartRepo{}, nil
}

func mockLoadIndexFile(idxFile repo.IndexFile) loadIndexFile {
	return func(path string) (*repo.IndexFile, error) {
		return &idxFile, nil
	}
}

type mockChartRepo struct {
	downloadFile func() (string, error)
}

func (m mockChartRepo) DownloadIndexFile() (string, error) {
	if m.downloadFile != nil {
		return m.downloadFile()
	}
	return "", nil
}
