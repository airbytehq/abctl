package helm

import (
	"testing"

	"github.com/airbytehq/abctl/internal/common"
	"github.com/google/go-cmp/cmp"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

func TestLocateChartFlag(t *testing.T) {
	expect := "chartFlagValue"
	c := LocateLatestAirbyteChart("", expect)
	if c != expect {
		t.Errorf("expected %q but got %q", expect, c)
	}
}

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
			exp: common.AirbyteRepoURL + "/example.test",
		},
		{
			name: "one non-release entry",
			entries: map[string]repo.ChartVersions{
				"airbyte": []*repo.ChartVersion{{
					Metadata: &chart.Metadata{Version: "1.2.3-alpha-df72e2940ca"},
					URLs:     []string{"example.test"},
				}},
			},
			exp: common.AirbyteChartName,
		},
		{
			name:    "no entries",
			entries: map[string]repo.ChartVersions{},
			exp:     common.AirbyteChartName,
		},
		{
			name: "one release entry with no URLs",
			entries: map[string]repo.ChartVersions{
				"airbyte": []*repo.ChartVersion{{
					Metadata: &chart.Metadata{Version: "1.2.3"},
					URLs:     []string{},
				}},
			},
			exp: common.AirbyteChartName,
		},
		{
			name: "one release entry with two URLs",
			entries: map[string]repo.ChartVersions{
				"airbyte": []*repo.ChartVersion{{
					Metadata: &chart.Metadata{Version: "1.2.3"},
					URLs:     []string{"one.test", "two.test"},
				}},
			},
			exp: common.AirbyteChartName,
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
			exp: common.AirbyteRepoURL + "/good.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultLoadIndexFile = mockLoadIndexFile(repo.IndexFile{Entries: tt.entries})
			act := LocateLatestAirbyteChart("", "")
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
