package service

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/airbytehq/abctl/internal/abctl"
	"github.com/airbytehq/abctl/internal/common"
	"github.com/airbytehq/abctl/internal/helm"
	"github.com/airbytehq/abctl/internal/helm/mock"
	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/k8s/k8stest"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/google/go-cmp/cmp"
	goHelm "github.com/mittwald/go-helm-client"
	"go.uber.org/mock/gomock"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
)

const (
	portTest            = 9999
	testAirbyteChartLoc = "https://airbytehq.github.io/helm-charts/airbyte-1.2.3.tgz"
)

func TestCommand_Install_HappyPath(t *testing.T) {
	valuesYaml := mustReadFile(t, "./testdata/test-edition.values.yaml")
	expNginxValues, _ := helm.BuildNginxValues(portTest)

	// This test covers the happy path for a successful Airbyte and Nginx install.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	helm := mock.NewMockClient(ctrl)
	helm.EXPECT().AddOrUpdateChartRepo(gomock.Any()).DoAndReturn(func(entry repo.Entry) error {
		switch entry.Name {
		case common.AirbyteRepoName:
			if d := cmp.Diff(common.AirbyteRepoURLv1, entry.URL); d != "" {
				t.Error("chart url mismatch", d)
			}
		case common.NginxRepoName:
			if d := cmp.Diff(common.NginxRepoURL, entry.URL); d != "" {
				t.Error("chart url mismatch", d)
			}
		default:
			t.Error("unexpected chart repo name", entry.Name)
		}
		return nil
	}).Times(2)
	helm.EXPECT().GetChart(gomock.Any(), gomock.Any()).DoAndReturn(func(name string, _ *action.ChartPathOptions) (*chart.Chart, string, error) {
		switch name {
		case testAirbyteChartLoc:
			return &chart.Chart{Metadata: &chart.Metadata{Version: "test.airbyte.version"}}, "", nil
		case common.NginxChartName:
			return &chart.Chart{Metadata: &chart.Metadata{Version: "test.nginx.version"}}, "", nil
		default:
			t.Error("unsupported chart name", name)
			return nil, "", errors.New("unexpected chart name")
		}
	}).Times(2)
	helm.EXPECT().GetRelease(gomock.Any()).DoAndReturn(func(name string) (*release.Release, error) {
		switch name {
		case common.AirbyteChartRelease:
			t.Error("should not have been called", name)
			return nil, errors.New("should not have been called")
		case common.NginxChartRelease:
			return nil, errors.New("not found")
		default:
			t.Error("unsupported chart name", name)
			return nil, errors.New("unexpected chart name")
		}
	}).AnyTimes()
	helm.EXPECT().InstallOrUpgradeChart(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, spec *goHelm.ChartSpec, opts *goHelm.GenericHelmOptions) (*release.Release, error) {
		switch spec.ReleaseName {
		case common.AirbyteChartRelease:
			exp := &goHelm.ChartSpec{
				ReleaseName:     common.AirbyteChartRelease,
				ChartName:       testAirbyteChartLoc,
				Namespace:       common.AirbyteNamespace,
				CreateNamespace: true,
				Wait:            true,
				Timeout:         60 * time.Minute,
				ValuesYaml:      valuesYaml,
			}
			if d := cmp.Diff(exp, spec); d != "" {
				t.Error("chart mismatch", d)
			}
			return &release.Release{
				Chart:     &chart.Chart{Metadata: &chart.Metadata{Version: "1.2.3.4"}},
				Name:      common.AirbyteChartRelease,
				Namespace: common.AirbyteNamespace,
				Version:   0,
			}, nil
		case common.NginxChartRelease:
			exp := &goHelm.ChartSpec{
				ReleaseName:     common.NginxChartRelease,
				ChartName:       common.NginxChartName,
				Namespace:       common.NginxNamespace,
				CreateNamespace: true,
				Wait:            true,
				Timeout:         60 * time.Minute,
				ValuesYaml:      expNginxValues,
			}
			if d := cmp.Diff(exp, spec); d != "" {
				t.Error("chart mismatch", d)
			}
			return &release.Release{
				Chart:     &chart.Chart{Metadata: &chart.Metadata{Version: "4.3.2.1"}},
				Name:      common.NginxChartRelease,
				Namespace: common.NginxNamespace,
				Version:   0,
			}, nil
		}
		t.Error("unexpected release name", spec.ReleaseName)
		return nil, errors.New("unexpected release name")
	}).Times(2)
	helm.EXPECT().UninstallReleaseByName(gomock.Any()).DoAndReturn(func(s string) error {
		if s != common.AirbyteChartRelease && s != common.NginxChartRelease {
			t.Error("unexpected uninstall release name", s)
		}
		return nil
	}).AnyTimes()

	k8sClient := k8stest.MockClient{
		FnIngressExists: func(ctx context.Context, namespace string, ingress string) bool {
			return false
		},
	}
	tel := telemetry.MockClient{}
	httpClient := mockHTTP{do: func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200}, nil
	}}
	svcMgr, err := NewManager(
		k8s.TestProvider,
		WithPortHTTP(portTest),
		WithHelmClient(helm),
		WithK8sClient(&k8sClient),
		WithTelemetryClient(&tel),
		WithHTTPClient(&httpClient),
		WithBrowserLauncher(func(url string) error { return nil }),
	)
	if err != nil {
		t.Fatal(err)
	}
	installOpts := &InstallOpts{
		HelmValuesYaml:  valuesYaml,
		AirbyteChartLoc: testAirbyteChartLoc,
	}
	if err := svcMgr.Install(context.Background(), installOpts); err != nil {
		t.Fatal(err)
	}
}

func TestCommand_Install_BadHelmState(t *testing.T) {
	valuesYaml := mustReadFile(t, "testdata/test-edition.values.yaml")

	// This test simulates a transient Helm stuck state that recovers on retry.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	helm := mock.NewMockClient(ctrl)
	// Expect both repos to be added.
	helm.EXPECT().AddOrUpdateChartRepo(gomock.Any()).Return(nil).Times(2)
	// Expect both charts to be fetched.
	helm.EXPECT().GetChart(gomock.Any(), gomock.Any()).DoAndReturn(func(name string, _ *action.ChartPathOptions) (*chart.Chart, string, error) {
		switch name {
		case testAirbyteChartLoc:
			return &chart.Chart{Metadata: &chart.Metadata{Version: "test.airbyte.version"}}, "", nil
		case common.NginxChartName:
			return &chart.Chart{Metadata: &chart.Metadata{Version: "test.nginx.version"}}, "", nil
		default:
			t.Error("unsupported chart name", name)
			return nil, "", errors.New("unexpected chart name")
		}
	}).Times(2)
	// Always return not found for GetRelease.
	helm.EXPECT().GetRelease(gomock.Any()).AnyTimes().Return(nil, errors.New("not found"))
	// First Airbyte install fails with errHelmStuck, triggering retry logic.
	helm.EXPECT().InstallOrUpgradeChart(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errHelmStuck)
	// Second Airbyte install and Nginx install both succeed.
	helm.EXPECT().InstallOrUpgradeChart(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, spec *goHelm.ChartSpec, opts *goHelm.GenericHelmOptions) (*release.Release, error) {
		switch spec.ReleaseName {
		case common.AirbyteChartRelease:
			return &release.Release{
				Chart:     &chart.Chart{Metadata: &chart.Metadata{Version: "1.2.3.4"}},
				Name:      common.AirbyteChartRelease,
				Namespace: common.AirbyteNamespace,
				Version:   0,
			}, nil
		case common.NginxChartRelease:
			return &release.Release{
				Chart:     &chart.Chart{Metadata: &chart.Metadata{Version: "4.3.2.1"}},
				Name:      common.NginxChartRelease,
				Namespace: common.NginxNamespace,
				Version:   0,
			}, nil
		}
		t.Error("unexpected release name", spec.ReleaseName)
		return nil, errors.New("unexpected release name")
	}).Times(2)
	helm.EXPECT().UninstallReleaseByName(gomock.Any()).AnyTimes().Return(nil)

	k8sClient := k8stest.MockClient{
		FnIngressExists: func(ctx context.Context, namespace string, ingress string) bool {
			return false
		},
	}
	tel := telemetry.MockClient{}
	httpClient := mockHTTP{do: func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200}, nil
	}}
	svcMgr, err := NewManager(
		k8s.TestProvider,
		WithPortHTTP(portTest),
		WithHelmClient(helm),
		WithK8sClient(&k8sClient),
		WithTelemetryClient(&tel),
		WithHTTPClient(&httpClient),
		WithBrowserLauncher(func(url string) error { return nil }),
	)
	if err != nil {
		t.Fatal(err)
	}
	installOpts := &InstallOpts{
		HelmValuesYaml:  valuesYaml,
		AirbyteChartLoc: testAirbyteChartLoc,
	}
	if err := svcMgr.Install(context.Background(), installOpts); err != nil {
		t.Fatal(err)
	}
}

func TestCommand_Install_BadHelmStatePersists(t *testing.T) {
	valuesYaml := mustReadFile(t, "testdata/test-edition.values.yaml")

	// This test simulates the case where the Helm stuck error persists for all retries.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	helm := mock.NewMockClient(ctrl)
	helm.EXPECT().AddOrUpdateChartRepo(gomock.Any()).AnyTimes().Return(nil)
	helm.EXPECT().GetChart(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(name string, _ *action.ChartPathOptions) (*chart.Chart, string, error) {
		switch name {
		case testAirbyteChartLoc:
			return &chart.Chart{Metadata: &chart.Metadata{Version: "test.airbyte.version"}}, "", nil
		case common.NginxChartName:
			return &chart.Chart{Metadata: &chart.Metadata{Version: "test.nginx.version"}}, "", nil
		default:
			t.Error("unsupported chart name", name)
			return nil, "", errors.New("unexpected chart name")
		}
	})
	helm.EXPECT().GetRelease(gomock.Any()).AnyTimes().Return(nil, errors.New("not found"))
	// Chain: error, error, error (simulate persistent failure)
	first := helm.EXPECT().InstallOrUpgradeChart(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("another operation (install/upgrade/rollback) is in progress")).Times(1)
	second := helm.EXPECT().InstallOrUpgradeChart(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("another operation (install/upgrade/rollback) is in progress")).Times(1).After(first)
	helm.EXPECT().InstallOrUpgradeChart(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("another operation (install/upgrade/rollback) is in progress")).Times(1).After(second)
	helm.EXPECT().UninstallReleaseByName(gomock.Any()).AnyTimes().Return(nil)

	k8sClient := k8stest.MockClient{
		FnIngressExists: func(ctx context.Context, namespace string, ingress string) bool {
			return false
		},
	}
	tel := telemetry.MockClient{}
	httpClient := mockHTTP{do: func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200}, nil
	}}
	svcMgr, err := NewManager(
		k8s.TestProvider,
		WithPortHTTP(portTest),
		WithHelmClient(helm),
		WithK8sClient(&k8sClient),
		WithTelemetryClient(&tel),
		WithHTTPClient(&httpClient),
		WithBrowserLauncher(func(url string) error { return nil }),
	)
	if err != nil {
		t.Fatal(err)
	}
	installOpts := &InstallOpts{
		HelmValuesYaml:  valuesYaml,
		AirbyteChartLoc: testAirbyteChartLoc,
	}
	if err := svcMgr.Install(context.Background(), installOpts); err == nil {
		t.Fatal("expected error")
	} else if !errors.Is(err, abctl.ErrHelmStuck) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommand_InstallError(t *testing.T) {
	testErr := errors.New("test error")
	valuesYaml := mustReadFile(t, "testdata/test-edition.values.yaml")

	// This test simulates a generic install error (not a Helm stuck error).
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	helm := mock.NewMockClient(ctrl)
	helm.EXPECT().AddOrUpdateChartRepo(gomock.Any()).AnyTimes().Return(nil)
	helm.EXPECT().GetChart(gomock.Any(), gomock.Any()).AnyTimes().Return(&chart.Chart{Metadata: &chart.Metadata{Version: "test.airbyte.version"}}, "", nil)
	helm.EXPECT().InstallOrUpgradeChart(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, testErr)

	k8sClient := k8stest.MockClient{
		FnIngressExists: func(ctx context.Context, namespace string, ingress string) bool {
			return false
		},
		FnLogsGet: func(ctx context.Context, namespace, name string) (string, error) {
			return "short", nil
		},
		FnPodList: func(ctx context.Context, namespace string) (*corev1.PodList, error) {
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
	svcMgr, err := NewManager(
		k8s.TestProvider,
		WithPortHTTP(portTest),
		WithHelmClient(helm),
		WithK8sClient(&k8sClient),
		WithTelemetryClient(&tel),
		WithHTTPClient(&httpClient),
		WithBrowserLauncher(func(url string) error { return nil }),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = svcMgr.Install(context.Background(), installOpts)
	expect := "unable to install airbyte chart: unable to install helm: test error"
	if expect != err.Error() {
		t.Errorf("expected %q but got %q", expect, err)
	}
}

func mustReadFile(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
