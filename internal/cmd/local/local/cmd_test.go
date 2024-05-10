package local

import (
	"context"
	"errors"
	"fmt"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/google/go-cmp/cmp"
	helmclient "github.com/mittwald/go-helm-client"
	"github.com/mittwald/go-helm-client/values"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	coreV1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"net/http"
	"testing"
	"time"
)

const portTest = 9999

func TestCommand_Install(t *testing.T) {
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
				Chart:     &chart.Chart{Metadata: &chart.Metadata{Version: "1.2.3.4"}},
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
				ValuesOptions:   values.Options{Values: []string{fmt.Sprintf("controller.service.ports.http=%d", portTest)}},
			},
			release: release.Release{
				Chart:     &chart.Chart{Metadata: &chart.Metadata{Version: "4.3.2.1"}},
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
		WithPortHTTP(portTest),
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

var _ k8s.Client = (*mockK8sClient)(nil)

type mockK8sClient struct {
	createIngress        func(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error
	existsIngress        func(ctx context.Context, namespace string, ingress string) bool
	updateIngress        func(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error
	existsNamespace      func(ctx context.Context, namespace string) bool
	deleteNamespace      func(ctx context.Context, namespace string) error
	createOrUpdateSecret func(ctx context.Context, namespace, name string, data map[string][]byte) error
	getService           func(ctx context.Context, namespace, name string) (*coreV1.Service, error)
	getServerVersion     func() (string, error)
}

func (m *mockK8sClient) IngressCreate(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
	return m.createIngress(ctx, namespace, ingress)
}

func (m *mockK8sClient) IngressExists(ctx context.Context, namespace string, ingress string) bool {
	return m.existsIngress(ctx, namespace, ingress)
}

func (m *mockK8sClient) IngressUpdate(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
	return m.updateIngress(ctx, namespace, ingress)
}

func (m *mockK8sClient) NamespaceExists(ctx context.Context, namespace string) bool {
	return m.existsNamespace(ctx, namespace)
}

func (m *mockK8sClient) NamespaceDelete(ctx context.Context, namespace string) error {
	return m.deleteNamespace(ctx, namespace)
}

func (m *mockK8sClient) SecretCreateOrUpdate(ctx context.Context, namespace, name string, data map[string][]byte) error {
	return m.createOrUpdateSecret(ctx, namespace, name, data)
}

func (m *mockK8sClient) ServiceGet(ctx context.Context, namespace, name string) (*coreV1.Service, error) {
	return m.getService(ctx, namespace, name)
}

func (m *mockK8sClient) ServerVersionGet() (string, error) {
	return m.getServerVersion()
}

var _ telemetry.Client = (*mockTelemetryClient)(nil)

type mockTelemetryClient struct {
	start   func(context.Context, telemetry.EventType) error
	success func(context.Context, telemetry.EventType) error
	failure func(context.Context, telemetry.EventType, error) error
	attr    func(key, val string)
}

func (m *mockTelemetryClient) Start(ctx context.Context, eventType telemetry.EventType) error {
	return m.start(ctx, eventType)
}

func (m *mockTelemetryClient) Success(ctx context.Context, eventType telemetry.EventType) error {
	return m.success(ctx, eventType)
}

func (m *mockTelemetryClient) Failure(ctx context.Context, eventType telemetry.EventType, err error) error {
	return m.failure(ctx, eventType, err)
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
