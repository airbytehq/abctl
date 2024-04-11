package local

import (
	"context"
	helmclient "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	networkingv1 "k8s.io/api/networking/v1"
)

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

var _ K8sClient = (*mockK8sClient)(nil)

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
