package local

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s/k8stest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	helmclient "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

const portTest = 9999
const testAirbyteChartLoc = "https://airbytehq.github.io/helm-charts/airbyte-1.2.3.tgz"

func testChartLocator(chartName, chartVersion, chartFlag string) string {
	if chartName == airbyteChartName && chartVersion == "" {
		return testAirbyteChartLoc
	}
	return chartName
}

func TestCommand_Install(t *testing.T) {
	expNginxValues, _ := getNginxValuesYaml(9999)
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
				ChartName:       testAirbyteChartLoc,
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
				ValuesYaml:      expNginxValues,
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
			case name == testAirbyteChartLoc:
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

		uninstallReleaseByName: func(s string) error {
			if d := cmp.Diff(expChart[expChartCnt].release.Name, s); d != "" {
				t.Error("release mismatch", d)
			}

			return nil
		},
	}

	k8sClient := k8stest.MockClient{
		FnServerVersionGet: func() (string, error) {
			return "test", nil
		},
		FnSecretCreateOrUpdate: func(ctx context.Context, secret corev1.Secret) error {
			return nil
		},
		FnIngressExists: func(ctx context.Context, namespace string, ingress string) bool {
			return false
		},
		FnIngressCreate: func(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
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
		WithChartLocator(testChartLocator),
	)

	if err != nil {
		t.Fatal(err)
	}

	if err := c.Install(context.Background(), InstallOpts{}); err != nil {
		t.Fatal(err)
	}
}

func TestCommand_Install_HelmValues(t *testing.T) {
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
	expNginxValues, _ := getNginxValuesYaml(9999)
	expChart := []struct {
		chart   helmclient.ChartSpec
		release release.Release
	}{
		{
			chart: helmclient.ChartSpec{
				ReleaseName:     airbyteChartRelease,
				ChartName:       testAirbyteChartLoc,
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
				ValuesYaml:      expNginxValues,
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
			case name == testAirbyteChartLoc:
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

		uninstallReleaseByName: func(s string) error {
			if d := cmp.Diff(expChart[expChartCnt].release.Name, s); d != "" {
				t.Error("release mismatch", d)
			}

			return nil
		},
	}

	k8sClient := k8stest.MockClient{
		FnServerVersionGet: func() (string, error) {
			return "test", nil
		},
		FnSecretCreateOrUpdate: func(ctx context.Context, secret corev1.Secret) error {
			return nil
		},
		FnIngressExists: func(ctx context.Context, namespace string, ingress string) bool {
			return false
		},
		FnIngressCreate: func(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
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
		WithChartLocator(testChartLocator),
	)

	if err != nil {
		t.Fatal(err)
	}

	helmValues := map[string]any{
		"global": map[string]any{
			"edition": "test",
		},
	}
	if err := c.Install(context.Background(), InstallOpts{HelmValues: helmValues}); err != nil {
		t.Fatal(err)
	}
}
