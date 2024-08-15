package local

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/airbytehq/abctl/internal/cmd/local/helm"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	helmclient "github.com/mittwald/go-helm-client"
	"github.com/mittwald/go-helm-client/values"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	coreV1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/watch"
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

	// userID is for telemetry tracking purposes
	userID := uuid.New()

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
				Timeout:         30 * time.Minute,
				ValuesYaml: `global:
    auth:
        enabled: "true"
    env_vars:
        AIRBYTE_INSTALLATION_ID: ` + userID.String() + `
    jobs:
        resources:
            limits:
                cpu: "3"
                memory: 4Gi
`,
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
				Timeout:         30 * time.Minute,
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

		getRelease: func(name string) (*release.Release, error) {
			switch {
			case name == airbyteChartRelease:
				t.Error("should not have been called", name)
				return nil, errors.New("should not have been called")
			case name == nginxChartRelease:
				return nil, errors.New("not found")
			default:
				t.Error("unsupported chart name", name)
				return nil, errors.New("unexpected chart name")
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
		serverVersionGet: func() (string, error) {
			return "test", nil
		},
		secretCreateOrUpdate: func(ctx context.Context, secret coreV1.Secret) error {
			return nil
		},
		ingressExists: func(ctx context.Context, namespace string, ingress string) bool {
			return false
		},
		ingressCreate: func(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
			return nil
		},
	}

	attrs := map[string]string{}
	tel := mockTelemetryClient{
		attr: func(key, val string) { attrs[key] = val },
		user: func() uuid.UUID { return userID },
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

	if err := c.Install(context.Background(), InstallOpts{}); err != nil {
		t.Fatal(err)
	}
}

func TestCommand_Install_ValuesFile(t *testing.T) {
	expChartRepoCnt := 0
	expChartRepo := []struct {
		name string
		url  string
	}{
		{name: airbyteRepoName, url: airbyteRepoURL},
		{name: nginxRepoName, url: nginxRepoURL},
	}

	// userID is for telemetry tracking purposes
	userID := uuid.New()

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
				Timeout:         30 * time.Minute,
				ValuesYaml: `global:
    auth:
        enabled: "true"
    edition: test
    env_vars:
        AIRBYTE_INSTALLATION_ID: ` + userID.String() + `
    jobs:
        resources:
            limits:
                cpu: "3"
                memory: 4Gi
`,
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
				Timeout:         30 * time.Minute,
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

		getRelease: func(name string) (*release.Release, error) {
			switch {
			case name == airbyteChartRelease:
				t.Error("should not have been called", name)
				return nil, errors.New("should not have been called")
			case name == nginxChartRelease:
				return nil, errors.New("not found")
			default:
				t.Error("unsupported chart name", name)
				return nil, errors.New("unexpected chart name")
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
		serverVersionGet: func() (string, error) {
			return "test", nil
		},
		secretCreateOrUpdate: func(ctx context.Context, secret coreV1.Secret) error {
			return nil
		},
		ingressExists: func(ctx context.Context, namespace string, ingress string) bool {
			return false
		},
		ingressCreate: func(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
			return nil
		},
	}

	attrs := map[string]string{}
	tel := mockTelemetryClient{
		attr: func(key, val string) { attrs[key] = val },
		user: func() uuid.UUID { return userID },
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

	if err := c.Install(context.Background(), InstallOpts{ValuesFile: "testdata/values.yml"}); err != nil {
		t.Fatal(err)
	}
}

func TestCommand_Install_InvalidValuesFile(t *testing.T) {
	c, err := New(
		k8s.TestProvider,
		WithPortHTTP(portTest),
		WithHelmClient(&mockHelmClient{}),
		WithK8sClient(&mockK8sClient{}),
		WithTelemetryClient(&mockTelemetryClient{}),
		WithHTTPClient(&mockHTTP{}),
		WithBrowserLauncher(func(url string) error {
			return nil
		}),
	)

	if err != nil {
		t.Fatal(err)
	}

	valuesFile := "testdata/dne.yml"

	err = c.Install(context.Background(), InstallOpts{ValuesFile: valuesFile})
	if err == nil {
		t.Fatal("expecting an error, received none")
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("unable to read values from yaml file '%s'", valuesFile)) {
		t.Error("unexpected error:", err)
	}

}

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
	ingressCreate               func(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error
	ingressExists               func(ctx context.Context, namespace string, ingress string) bool
	ingressUpdate               func(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error
	namespaceCreate             func(ctx context.Context, namespace string) error
	namespaceExists             func(ctx context.Context, namespace string) bool
	namespaceDelete             func(ctx context.Context, namespace string) error
	persistentVolumeCreate      func(ctx context.Context, namespace, name string) error
	persistentVolumeExists      func(ctx context.Context, namespace, name string) bool
	persistentVolumeDelete      func(ctx context.Context, namespace, name string) error
	persistentVolumeClaimCreate func(ctx context.Context, namespace, name, volumeName string) error
	persistentVolumeClaimExists func(ctx context.Context, namespace, name, volumeName string) bool
	persistentVolumeClaimDelete func(ctx context.Context, namespace, name, volumeName string) error
	secretCreateOrUpdate        func(ctx context.Context, secret coreV1.Secret) error
	secretGet                   func(ctx context.Context, namespace, name string) (*coreV1.Secret, error)
	serviceGet                  func(ctx context.Context, namespace, name string) (*coreV1.Service, error)
	serverVersionGet            func() (string, error)
	eventsWatch                 func(ctx context.Context, namespace string) (watch.Interface, error)
	logsGet                     func(ctx context.Context, namespace string, name string) (string, error)
}

func (m *mockK8sClient) IngressCreate(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
	if m.ingressCreate != nil {
		return m.ingressCreate(ctx, namespace, ingress)
	}
	return nil
}

func (m *mockK8sClient) IngressExists(ctx context.Context, namespace string, ingress string) bool {
	if m.ingressExists != nil {
		return m.ingressExists(ctx, namespace, ingress)
	}
	return true
}

func (m *mockK8sClient) IngressUpdate(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
	if m.ingressUpdate != nil {
		return m.ingressUpdate(ctx, namespace, ingress)
	}
	return nil
}

func (m *mockK8sClient) NamespaceCreate(ctx context.Context, namespace string) error {
	if m.namespaceCreate != nil {
		return m.namespaceCreate(ctx, namespace)
	}
	return nil
}

func (m *mockK8sClient) NamespaceExists(ctx context.Context, namespace string) bool {
	if m.namespaceExists != nil {
		return m.namespaceExists(ctx, namespace)
	}
	return true
}

func (m *mockK8sClient) NamespaceDelete(ctx context.Context, namespace string) error {
	if m.namespaceDelete != nil {
		return m.namespaceDelete(ctx, namespace)
	}
	return nil
}

func (m *mockK8sClient) PersistentVolumeCreate(ctx context.Context, namespace, name string) error {
	if m.persistentVolumeCreate != nil {
		return m.persistentVolumeCreate(ctx, namespace, name)
	}
	return nil
}
func (m *mockK8sClient) PersistentVolumeExists(ctx context.Context, namespace, name string) bool {
	if m.persistentVolumeExists != nil {
		return m.persistentVolumeExists(ctx, namespace, name)
	}
	return true
}
func (m *mockK8sClient) PersistentVolumeDelete(ctx context.Context, namespace, name string) error {
	if m.persistentVolumeDelete != nil {
		return m.persistentVolumeDelete(ctx, namespace, name)
	}
	return nil
}

func (m *mockK8sClient) PersistentVolumeClaimCreate(ctx context.Context, namespace, name, volumeName string) error {
	if m.persistentVolumeClaimCreate != nil {
		return m.persistentVolumeClaimCreate(ctx, namespace, name, volumeName)
	}
	return nil
}
func (m *mockK8sClient) PersistentVolumeClaimExists(ctx context.Context, namespace, name, volumeName string) bool {
	if m.persistentVolumeClaimExists != nil {
		return m.persistentVolumeClaimExists(ctx, namespace, name, volumeName)
	}
	return true
}
func (m *mockK8sClient) PersistentVolumeClaimDelete(ctx context.Context, namespace, name, volumeName string) error {
	if m.persistentVolumeClaimDelete != nil {
		return m.persistentVolumeClaimDelete(ctx, namespace, name, volumeName)
	}
	return nil
}

func (m *mockK8sClient) SecretCreateOrUpdate(ctx context.Context, secret coreV1.Secret) error {
	if m.secretCreateOrUpdate != nil {
		return m.secretCreateOrUpdate(ctx, secret)
	}

	return nil
}

func (m *mockK8sClient) SecretGet(ctx context.Context, namespace, name string) (*coreV1.Secret, error) {
	if m.secretGet != nil {
		return m.secretGet(ctx, namespace, name)
	}

	return nil, nil
}

func (m *mockK8sClient) ServiceGet(ctx context.Context, namespace, name string) (*coreV1.Service, error) {
	return m.serviceGet(ctx, namespace, name)
}

func (m *mockK8sClient) ServerVersionGet() (string, error) {
	if m.serverVersionGet != nil {
		return m.serverVersionGet()
	}
	return "test", nil
}

func (m *mockK8sClient) EventsWatch(ctx context.Context, namespace string) (watch.Interface, error) {
	if m.eventsWatch == nil {
		return watch.NewFake(), nil
	}
	return m.eventsWatch(ctx, namespace)
}

func (m *mockK8sClient) LogsGet(ctx context.Context, namespace string, name string) (string, error) {
	if m.logsGet == nil {
		return "LogsGet called", nil
	}
	return m.logsGet(ctx, namespace, name)
}

var _ telemetry.Client = (*mockTelemetryClient)(nil)

type mockTelemetryClient struct {
	start   func(context.Context, telemetry.EventType) error
	success func(context.Context, telemetry.EventType) error
	failure func(context.Context, telemetry.EventType, error) error
	attr    func(key, val string)
	user    func() uuid.UUID
	wrap    func(context.Context, telemetry.EventType, func() error) error
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
	if m.attr != nil {
		m.attr(key, val)
	}
}

func (m *mockTelemetryClient) User() uuid.UUID {
	if m.user == nil {
		return uuid.Nil
	}
	return m.user()
}

func (m *mockTelemetryClient) Wrap(ctx context.Context, et telemetry.EventType, f func() error) error {
	return m.wrap(ctx, et, f)
}

var _ HTTPClient = (*mockHTTP)(nil)

type mockHTTP struct {
	do func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTP) Do(req *http.Request) (*http.Response, error) {
	return m.do(req)
}
