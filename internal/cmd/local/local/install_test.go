package local

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/airbytehq/abctl/internal/cmd/local/helm"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s/k8stest"
	"github.com/airbytehq/abctl/internal/common"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/google/go-cmp/cmp"
	helmclient "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
)

const portTest = 9999
const testAirbyteChartLoc = "https://airbytehq.github.io/helm-charts/airbyte-1.2.3.tgz"

func TestCommand_Install(t *testing.T) {
	valuesYaml := mustReadFile(t, "testdata/test-edition.values.yaml")
	expChartRepoCnt := 0
	expChartRepo := []struct {
		name string
		url  string
	}{
		{name: common.AirbyteRepoName, url: common.AirbyteRepoURL},
		{name: common.NginxRepoName, url: common.NginxRepoURL},
	}

	expChartCnt := 0
	expNginxValues, _ := helm.BuildNginxValues(9999)
	expChart := []struct {
		chart   helmclient.ChartSpec
		release release.Release
	}{
		{
			chart: helmclient.ChartSpec{
				ReleaseName:     common.AirbyteChartRelease,
				ChartName:       testAirbyteChartLoc,
				Namespace:       common.AirbyteNamespace,
				CreateNamespace: true,
				Wait:            true,
				Timeout:         60 * time.Minute,
				ValuesYaml:      valuesYaml,
			},
			release: release.Release{
				Chart:     &chart.Chart{Metadata: &chart.Metadata{Version: "1.2.3.4"}},
				Name:      common.AirbyteChartRelease,
				Namespace: common.AirbyteNamespace,
				Version:   0,
			},
		},
		{
			chart: helmclient.ChartSpec{
				ReleaseName:     common.NginxChartRelease,
				ChartName:       common.NginxChartName,
				Namespace:       common.NginxNamespace,
				CreateNamespace: true,
				Wait:            true,
				Timeout:         60 * time.Minute,
				ValuesYaml:      expNginxValues,
			},
			release: release.Release{
				Chart:     &chart.Chart{Metadata: &chart.Metadata{Version: "4.3.2.1"}},
				Name:      common.NginxChartRelease,
				Namespace: common.NginxNamespace,
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
			case name == common.NginxChartName:
				return &chart.Chart{Metadata: &chart.Metadata{Version: "test.nginx.version"}}, "", nil
			default:
				t.Error("unsupported chart name", name)
				return nil, "", errors.New("unexpected chart name")
			}
		},

		getRelease: func(name string) (*release.Release, error) {
			switch {
			case name == common.AirbyteChartRelease:
				t.Error("should not have been called", name)
				return nil, errors.New("should not have been called")
			case name == common.NginxChartRelease:
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
		FnIngressExists: func(ctx context.Context, namespace string, ingress string) bool {
			return false
		},
	}

	tel := telemetry.MockClient{}

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

	installOpts := &InstallOpts{
		HelmValuesYaml:  valuesYaml,
		AirbyteChartLoc: testAirbyteChartLoc,
	}
	if err := c.Install(context.Background(), installOpts); err != nil {
		t.Fatal(err)
	}
}

func TestCommand_InstallError(t *testing.T) {
	testErr := errors.New("test error")
	valuesYaml := mustReadFile(t, "testdata/test-edition.values.yaml")

	helm := mockHelmClient{
		installOrUpgradeChart: func(ctx context.Context, spec *helmclient.ChartSpec, opts *helmclient.GenericHelmOptions) (*release.Release, error) {
			return nil, testErr
		},
		getChart: func(name string, _ *action.ChartPathOptions) (*chart.Chart, string, error) {
			return &chart.Chart{Metadata: &chart.Metadata{Version: "test.airbyte.version"}}, "", nil
		},
	}

	k8sClient := k8stest.MockClient{
		FnIngressExists: func(ctx context.Context, namespace string, ingress string) bool {
			return false
		},
		FnLogsGet: func(ctx context.Context, namespace, name string) (string, error) {
			return "short", nil
		},
		FnPodList: func(ctx context.Context, namespace string) (*corev1.PodList, error) {
			// embedded structs make it easier to set fields this way
			pod := corev1.Pod{}
			pod.Name = "test-pod-1"
			pod.Status.Phase = corev1.PodFailed
			pod.Status.Reason = "test-reason"
			return &corev1.PodList{Items: []corev1.Pod{pod}}, nil
		},
	}

	tel := telemetry.MockClient{}

	httpClient := mockHTTP{do: func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200}, nil
	}}
	installOpts := &InstallOpts{
		HelmValuesYaml:  valuesYaml,
		AirbyteChartLoc: testAirbyteChartLoc,
	}

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
	err = c.Install(context.Background(), installOpts)
	expect := "unable to install airbyte chart:\npod test-pod-1: unknown"
	if expect != err.Error() {
		t.Errorf("expected %q but got %q", expect, err)
	}
}

func mustReadFile(t *testing.T, name string) string {
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
