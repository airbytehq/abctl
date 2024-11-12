package local

import (
	"context"
	"net/http"

	"github.com/airbytehq/abctl/internal/cmd/local/helm"
	helmclient "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
)

// ---
// only mocks below here
// ---
var _ helm.Client = (*mockHelmClient)(nil)

type mockHelmClient struct {
	addOrUpdateChartRepo   func(entry repo.Entry) error
	getChart               func(string, *action.ChartPathOptions) (*chart.Chart, string, error)
	getRelease             func(name string) (*release.Release, error)
	installOrUpgradeChart  func(ctx context.Context, spec *helmclient.ChartSpec, opts *helmclient.GenericHelmOptions) (*release.Release, error)
	uninstallReleaseByName func(s string) error
}

func (m *mockHelmClient) AddOrUpdateChartRepo(entry repo.Entry) error {
	if m.addOrUpdateChartRepo == nil {
		return nil
	}
	return m.addOrUpdateChartRepo(entry)
}

func (m *mockHelmClient) GetChart(s string, options *action.ChartPathOptions) (*chart.Chart, string, error) {
	return m.getChart(s, options)
}

func (m *mockHelmClient) GetRelease(name string) (*release.Release, error) {
	return m.getRelease(name)
}

func (m *mockHelmClient) InstallOrUpgradeChart(ctx context.Context, spec *helmclient.ChartSpec, opts *helmclient.GenericHelmOptions) (*release.Release, error) {
	return m.installOrUpgradeChart(ctx, spec, opts)
}

func (m *mockHelmClient) UninstallReleaseByName(s string) error {
	return m.uninstallReleaseByName(s)
}

func (m *mockHelmClient) TemplateChart(spec *helmclient.ChartSpec, options *helmclient.HelmTemplateOptions) ([]byte, error) {
	return nil, nil
}

var _ HTTPClient = (*mockHTTP)(nil)

type mockHTTP struct {
	do func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTP) Do(req *http.Request) (*http.Response, error) {
	return m.do(req)
}
