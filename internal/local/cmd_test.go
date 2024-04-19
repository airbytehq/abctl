package local

import (
	"context"
	"errors"
	"github.com/airbytehq/abctl/internal/local/k8s"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/docker/docker/api/types"
	"github.com/google/go-cmp/cmp"
	helmclient "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	networkingv1 "k8s.io/api/networking/v1"
	"net/http"
	"testing"
	"time"
)

func TestCommand_Install(t *testing.T) {
	docker := mockDockerClient{
		serverVersion: func(ctx context.Context) (types.Version, error) {
			return types.Version{}, nil
		},
	}

	expChartRepoCnt := 0
	expChartRepo := []struct {
		name string
		url  string
	}{
		{name: airbyteRepoName, url: airbyteRepoURL},
		{name: nginxRepoName, url: nginxRepoURL},
	}
	expChartCnt := 0
	expChart := []struct {
		chart   helmclient.ChartSpec
		release release.Release
	}{
		{
			chart: helmclient.ChartSpec{
				ReleaseName:     airbyteChartRelease,
				ChartName:       airbyteChartName,
				Namespace:       airbyteNamespace,
				CreateNamespace: true,
				Wait:            true,
				Timeout:         10 * time.Minute,
			},
			release: release.Release{
				Name:      airbyteChartRelease,
				Namespace: airbyteNamespace,
				Version:   0,
			},
		},
		{
			chart: helmclient.ChartSpec{
				ReleaseName:     nginxChartRelease,
				ChartName:       nginxChartName,
				Namespace:       nginxNamespace,
				CreateNamespace: true,
				Wait:            true,
				Timeout:         10 * time.Minute,
			},
			release: release.Release{
				Name:      nginxChartRelease,
				Namespace: nginxNamespace,
				Version:   0,
			},
		},
	}
	helm := mockHelmClient{
		addOrUpdateChartRepo: func(entry repo.Entry) error {
			if d := cmp.Diff(expChartRepo[expChartRepoCnt].name, entry.Name); d != "" {
				t.Error("chart name mismatch", d)
			}
			if d := cmp.Diff(expChartRepo[expChartRepoCnt].url, entry.URL); d != "" {
				t.Error("chart url mismatch", d)
			}

			expChartRepoCnt++

			return nil
		},

		getChart: func(name string, _ *action.ChartPathOptions) (*chart.Chart, string, error) {
			switch {
			case name == airbyteChartName:
				return &chart.Chart{Metadata: &chart.Metadata{Version: "test.airbyte.version"}}, "", nil
			case name == nginxChartName:
				return &chart.Chart{Metadata: &chart.Metadata{Version: "test.nginx.version"}}, "", nil
			default:
				t.Error("unsupported chart name", name)
				return nil, "", errors.New("unexpected chart name")
			}
		},

		installOrUpgradeChart: func(ctx context.Context, spec *helmclient.ChartSpec, opts *helmclient.GenericHelmOptions) (*release.Release, error) {
			if d := cmp.Diff(&expChart[expChartCnt].chart, spec); d != "" {
				t.Error("chart mismatch", d)
			}

			defer func() { expChartCnt++ }()

			return &expChart[expChartCnt].release, nil
		},
	}

	k8sClient := mockK8sClient{
		getServerVersion: func() (string, error) {
			return "test", nil
		},
		createOrUpdateSecret: func(ctx context.Context, namespace, name string, data map[string][]byte) error {
			return nil
		},
		existsIngress: func(ctx context.Context, namespace string, ingress string) bool {
			return false
		},
		createIngress: func(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
			return nil
		},
	}

	attrs := map[string]string{}
	tel := mockTelemetryClient{
		attr: func(key, val string) {
			attrs[key] = val
		},
	}

	httpClient := mockHTTP{do: func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200}, nil
	}}

	c, err := New(
		k8s.TestProvider,
		WithDockerClient(&docker),
		WithHelmClient(&helm),
		WithK8sClient(&k8sClient),
		WithTelemetryClient(&tel),
		WithHTTPClient(&httpClient),
		WithBrowserLauncher(func(url string) error {
			return nil
		}),
	)

	if err != nil {
		t.Fatal(err)
	}

	if err := c.Install(context.Background(), "user", "pass"); err != nil {
		t.Fatal(err)
	}
}

// ---
// only mocks below here
// ---
var _ HelmClient = (*mockHelmClient)(nil)

type mockHelmClient struct {
	addOrUpdateChartRepo   func(entry repo.Entry) error
	getChart               func(string, *action.ChartPathOptions) (*chart.Chart, string, error)
	getRelease             func(name string) (*release.Release, error)
	installOrUpgradeChart  func(ctx context.Context, spec *helmclient.ChartSpec, opts *helmclient.GenericHelmOptions) (*release.Release, error)
	uninstallReleaseByName func(s string) error
}

func (m *mockHelmClient) AddOrUpdateChartRepo(entry repo.Entry) error {
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

var _ k8s.K8sClient = (*mockK8sClient)(nil)

type mockK8sClient struct {
	createIngress        func(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error
	existsIngress        func(ctx context.Context, namespace string, ingress string) bool
	updateIngress        func(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error
	existsNamespace      func(ctx context.Context, namespace string) bool
	deleteNamespace      func(ctx context.Context, namespace string) error
	createOrUpdateSecret func(ctx context.Context, namespace, name string, data map[string][]byte) error
	getServerVersion     func() (string, error)
}

func (m *mockK8sClient) CreateIngress(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
	return m.createIngress(ctx, namespace, ingress)
}

func (m *mockK8sClient) ExistsIngress(ctx context.Context, namespace string, ingress string) bool {
	return m.existsIngress(ctx, namespace, ingress)
}

func (m *mockK8sClient) UpdateIngress(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
	return m.updateIngress(ctx, namespace, ingress)
}

func (m *mockK8sClient) ExistsNamespace(ctx context.Context, namespace string) bool {
	return m.existsNamespace(ctx, namespace)
}

func (m *mockK8sClient) DeleteNamespace(ctx context.Context, namespace string) error {
	return m.deleteNamespace(ctx, namespace)
}

func (m *mockK8sClient) CreateOrUpdateSecret(ctx context.Context, namespace, name string, data map[string][]byte) error {
	return m.createOrUpdateSecret(ctx, namespace, name, data)
}

func (m *mockK8sClient) GetServerVersion() (string, error) {
	return m.getServerVersion()
}

var _ DockerClient = (*mockDockerClient)(nil)

type mockDockerClient struct {
	serverVersion func(ctx context.Context) (types.Version, error)
}

func (m *mockDockerClient) ServerVersion(ctx context.Context) (types.Version, error) {
	return m.serverVersion(ctx)
}

var _ telemetry.Client = (*mockTelemetryClient)(nil)

type mockTelemetryClient struct {
	start   func(eventType telemetry.EventType) error
	success func(eventType telemetry.EventType) error
	failure func(eventType telemetry.EventType, err error) error
	attr    func(key, val string)
}

func (m *mockTelemetryClient) Start(eventType telemetry.EventType) error {
	return m.start(eventType)
}

func (m *mockTelemetryClient) Success(eventType telemetry.EventType) error {
	return m.success(eventType)
}

func (m *mockTelemetryClient) Failure(eventType telemetry.EventType, err error) error {
	return m.failure(eventType, err)
}

func (m *mockTelemetryClient) Attr(key, val string) {
	m.attr(key, val)
}

var _ HTTPClient = (*mockHTTP)(nil)

type mockHTTP struct {
	do func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTP) Do(req *http.Request) (*http.Response, error) {
	return m.do(req)
}
