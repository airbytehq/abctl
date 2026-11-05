package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/airbytehq/abctl/internal/airbox"
	airboxmock "github.com/airbytehq/abctl/internal/airbox/mock"
	"github.com/airbytehq/abctl/internal/api"
	apimock "github.com/airbytehq/abctl/internal/api/mock"
	"github.com/airbytehq/abctl/internal/helm"
	helmmock "github.com/airbytehq/abctl/internal/helm/mock"
	"github.com/airbytehq/abctl/internal/k8s"
	uimock "github.com/airbytehq/abctl/internal/ui/mock"
	"github.com/alecthomas/kong"
	goHelm "github.com/mittwald/go-helm-client"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/release"
)

func componentValuesFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "values.yaml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0600))
	return path
}

func TestComponentValuesAndFlags(t *testing.T) {
	first := componentValuesFile(t, "components:\n  cloudMcp:\n    enabled: true\n    cluster: primary\n    egress: {ports: [443]}\n")
	second := componentValuesFile(t, "components:\n  cloudMcp:\n    enabled: false\n    egress: {ports: []}\n")
	values, components, err := loadComponentValues([]string{first, second})
	require.NoError(t, err)
	require.False(t, components["cloudMcp"].Enabled)
	require.Equal(t, "primary", components["cloudMcp"].Cluster)
	require.Empty(t, values["components"].(map[string]any)["cloudMcp"].(map[string]any)["egress"].(map[string]any)["ports"])
	var root struct {
		Dataplane DataplaneCmd `cmd:""`
	}
	parser, err := kong.New(&root)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"dataplane", "--values", first, "--values", second, "--cluster-context", "primary=ctx-a", "--cluster-context", "secondary=ctx-b"})
	require.NoError(t, err)
	require.Equal(t, []string{first, second}, root.Dataplane.Values)
	require.Equal(t, map[string]string{"primary": "ctx-a", "secondary": "ctx-b"}, root.Dataplane.ClusterContext)
	for _, contents := range []string{"", "{}", "components: {}", "components: null"} {
		_, components, err := loadComponentValues([]string{componentValuesFile(t, contents)})
		require.NoError(t, err)
		require.Empty(t, components)
	}
	for _, contents := range []string{"[invalid", "- not-a-map", "components: []", "components: {x: {enabled: definitely}}", "components: {x: {enabled: true}}", "components: {x: {enabled: true, cluster: p, publicIngress: {host: 'https://bad'}}}"} {
		_, _, err := loadComponentValues([]string{componentValuesFile(t, contents)})
		require.Error(t, err, contents)
	}
	_, _, err = loadComponentValues([]string{filepath.Join(t.TempDir(), "missing")})
	require.ErrorContains(t, err, "cannot be read")
}

type cloudConfigFunc func(context.Context, string, string, string, string) error

func (f cloudConfigFunc) ConfigureCloudMCP(ctx context.Context, namespace, release, id, secret string) error {
	return f(ctx, namespace, release, id, secret)
}

type componentFixture struct {
	store   *airbox.FileConfigStore
	context *airbox.Context
	api     *apimock.MockService
	ui      *uimock.MockProvider
	ctrl    *gomock.Controller
	events  []string
	logs    []string
	values  map[string]map[string]any
	waits   map[string]bool
}

func savedComponentInstallation() *airbox.DataplaneInstallation {
	return &airbox.DataplaneInstallation{DataplaneID: "dp", ClientID: "dp-client", ClientSecret: "test-dp-only", Clusters: map[string]airbox.ComponentCluster{"primary": {Context: "ctx-a", Namespace: "airbyte"}}, Registered: map[string]string{}}
}

func componentTestFixture(t *testing.T, previous *airbox.DataplaneInstallation) *componentFixture {
	t.Helper()
	t.Setenv("AIRBOXCONFIG", filepath.Join(t.TempDir(), "config"))
	f := &componentFixture{store: &airbox.FileConfigStore{}, ctrl: gomock.NewController(t), values: map[string]map[string]any{}, waits: map[string]bool{}}
	f.api = apimock.NewMockService(f.ctrl)
	f.ui = uimock.NewMockProvider(f.ctrl)
	f.ui.EXPECT().RunWithSpinner(gomock.Any(), gomock.Any()).DoAndReturn(func(_ string, operation func() error) error { return operation() }).AnyTimes()
	f.ui.EXPECT().ShowSuccess(gomock.Any()).AnyTimes()
	f.ui.EXPECT().NewLine().AnyTimes()
	f.ui.EXPECT().ShowInfo(gomock.Any()).Do(func(message string) { f.logs = append(f.logs, message) }).AnyTimes()
	current := airbox.Context{OrganizationID: "org", Edition: "cloud", AirbyteURL: "https://cloud.example.com", AirbyteAPIURL: "https://api.example.com", Auth: airbox.NewAuthWithOAuth2("test-client", "test-login-only"), DataplaneInstallations: map[string]*airbox.DataplaneInstallation{}}
	if previous != nil {
		current.DataplaneInstallations[installationKey("org", "install")] = previous
	}
	require.NoError(t, f.store.Save(&airbox.Config{CurrentContext: "cloud", Contexts: []airbox.NamedContext{{Name: "cloud", Context: current}}}))
	config, err := f.store.Load()
	require.NoError(t, err)
	f.context, err = config.GetCurrentContext()
	require.NoError(t, err)
	return f
}

func (f *componentFixture) helmFactory(t *testing.T, fail bool) helm.Factory {
	return func(kubeconfig, kubeContext, namespace string) (goHelm.Client, error) {
		require.Equal(t, "airbyte", namespace)
		client := helmmock.NewMockClient(f.ctrl)
		client.EXPECT().AddOrUpdateChartRepo(gomock.Any()).Return(nil)
		client.EXPECT().InstallOrUpgradeChart(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, spec *goHelm.ChartSpec, _ *goHelm.GenericHelmOptions) (*release.Release, error) {
			f.events = append(f.events, "helm:"+kubeContext)
			var values map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(spec.ValuesYaml), &values))
			f.values[kubeContext] = values
			f.waits[kubeContext] = spec.Wait
			require.Equal(t, spec.Wait, spec.Atomic)
			require.Equal(t, "install", spec.ReleaseName)
			if fail {
				return nil, errors.New("helm rejected")
			}
			return &release.Release{}, nil
		})
		return client, nil
	}
}

func (f *componentFixture) persisted(t *testing.T) *airbox.DataplaneInstallation {
	config, err := f.store.Load()
	require.NoError(t, err)
	current, err := config.GetCurrentContext()
	require.NoError(t, err)
	return current.DataplaneInstallations[installationKey("org", "install")]
}

func TestInstallComponentsMultipleClustersAndRegistration(t *testing.T) {
	f := componentTestFixture(t, nil)
	file := componentValuesFile(t, `componentsNamespace: custom-agents
components:
  cloudMcp: {enabled: true, cluster: primary, publicIngress: {host: mcp.example.com}}
  mtxApi: {enabled: true, cluster: secondary, publicIngress: {host: mtx.example.com}}
  aguiFrontend: {enabled: true, cluster: secondary, publicIngress: {host: agui.example.com}}
  aguiBackend: {enabled: true, cluster: primary, publicIngress: {host: backend.example.com}}
  directConnect: {enabled: true, cluster: primary}
  sourceConnect: {enabled: true, cluster: secondary}
  custom: {enabled: true, cluster: secondary, publicIngress: {host: custom.example.com}}
`)
	f.api.EXPECT().CreateDataplane(gomock.Any(), api.CreateDataplaneRequest{Name: "install", RegionID: "region", OrganizationID: "org", Enabled: true}).Return(&api.CreateDataplaneResponse{DataplaneID: "dp", ClientID: "dp-client", ClientSecret: "test-dp-only"}, nil)
	for _, registration := range []api.RegisterComponentRequest{{DataplaneID: "dp", Component: "agui", URL: "https://agui.example.com"}, {DataplaneID: "dp", Component: "cloud_mcp", URL: "https://mcp.example.com"}, {DataplaneID: "dp", Component: "mtx_api", URL: "https://mtx.example.com"}} {
		f.api.EXPECT().RegisterComponent(gomock.Any(), registration).DoAndReturn(func(_ context.Context, req api.RegisterComponentRequest) (*api.ComponentCredentials, error) {
			f.events = append(f.events, "register:"+req.Component)
			return &api.ComponentCredentials{ClientID: "mcp-client", ClientSecret: "rotated-mcp-only"}, nil
		})
	}
	cmd := &DataplaneCmd{Namespace: "airbyte", ChartVersion: "2.1.0", Values: []string{file}, ClusterContext: map[string]string{"primary": "ctx-a", "secondary": "ctx-b", "unused": "ctx-unused"}, WithKindCluster: true}
	cmd.componentClientFactory = func(config, kubeContext string) (k8s.ComponentClient, error) {
		require.Equal(t, "ctx-a", kubeContext)
		return cloudConfigFunc(func(_ context.Context, namespace, release, id, secret string) error {
			require.Equal(t, "custom-agents", namespace)
			require.Equal(t, "install", release)
			require.Equal(t, "mcp-client", id)
			require.Equal(t, "rotated-mcp-only", secret)
			f.events = append(f.events, "configure-cloud")
			return nil
		}), nil
	}
	require.NoError(t, cmd.installComponents(context.Background(), f.store, f.api, f.helmFactory(t, false), nil, f.ui, f.context, "install", "region"))
	require.Equal(t, []string{"helm:ctx-a", "helm:ctx-b", "register:agui", "register:cloud_mcp", "configure-cloud", "register:mtx_api"}, f.events)
	require.Equal(t, "primary", f.values["ctx-a"]["targetCluster"])
	require.Equal(t, "secondary", f.values["ctx-b"]["targetCluster"])
	require.False(t, f.waits["ctx-a"])
	require.True(t, f.waits["ctx-b"])
	require.Len(t, f.logs, 4)
	for _, key := range []string{"aguiBackend", "directConnect", "sourceConnect", "custom"} {
		count := 0
		for _, line := range f.logs {
			if strings.Contains(line, "Component "+key+" ") {
				count++
			}
		}
		require.Equal(t, 1, count, key)
	}
	state := f.persisted(t)
	require.Equal(t, map[string]string{"cloudMcp": "cloud_mcp", "mtxApi": "mtx_api", "aguiFrontend": "agui"}, state.Registered)
	require.Equal(t, "dp", state.DataplaneID)
	data, err := os.ReadFile(f.store.GetPath())
	require.NoError(t, err)
	require.NotContains(t, string(data), "rotated-mcp-only")
	info, err := os.Stat(f.store.GetPath())
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestInstallComponentsRemovalAndStableIdentity(t *testing.T) {
	previous := savedComponentInstallation()
	previous.Registered = map[string]string{"cloudMcp": "cloud_mcp", "aguiFrontend": "agui"}
	f := componentTestFixture(t, previous)
	for _, kind := range []string{"cloud_mcp", "agui"} {
		f.api.EXPECT().UnregisterComponent(gomock.Any(), api.UnregisterComponentRequest{DataplaneID: "dp", Component: kind}).DoAndReturn(func(_ context.Context, req api.UnregisterComponentRequest) error {
			f.events = append(f.events, "unregister:"+req.Component)
			return nil
		})
	}
	cmd := &DataplaneCmd{Namespace: "airbyte", ChartVersion: "2.1.0", Values: []string{componentValuesFile(t, `components: {cloudMcp: {enabled: false}}`)}}
	require.NoError(t, cmd.installComponents(context.Background(), f.store, f.api, f.helmFactory(t, false), nil, f.ui, f.context, "install", "unused-region"))
	require.Equal(t, "dp", f.values["ctx-a"]["dataPlane"].(map[string]any)["id"])
	require.Empty(t, f.persisted(t).Registered)
	require.Equal(t, []string{"helm:ctx-a", "unregister:agui", "unregister:cloud_mcp"}, f.events)
}

func TestInstallEmptyComponentOverlay(t *testing.T) {
	f := componentTestFixture(t, nil)
	f.api.EXPECT().CreateDataplane(gomock.Any(), gomock.Any()).Return(&api.CreateDataplaneResponse{DataplaneID: "dp", ClientID: "dp-client", ClientSecret: "test-dp-only"}, nil)
	cmd := &DataplaneCmd{Namespace: "airbyte", ChartVersion: "2.1.0", Values: []string{componentValuesFile(t, "")}, ClusterContext: map[string]string{"primary": "ctx-a"}}
	require.NoError(t, cmd.installComponents(context.Background(), f.store, f.api, f.helmFactory(t, false), nil, f.ui, f.context, "install", "region"))
	require.Empty(t, f.persisted(t).Registered)
	require.True(t, f.waits["ctx-a"])
}

func TestComponentDataplaneCreationErrors(t *testing.T) {
	for _, failure := range []string{"API error", "incomplete credentials", "Helm factory"} {
		t.Run(failure, func(t *testing.T) {
			f := componentTestFixture(t, nil)
			cmd := &DataplaneCmd{Namespace: "airbyte", ChartVersion: "2.1.0", Values: []string{componentValuesFile(t, "")}, ClusterContext: map[string]string{"primary": "ctx-a"}}
			factory := func(string, string, string) (goHelm.Client, error) {
				if failure == "Helm factory" {
					return nil, errors.New("context unavailable")
				}
				return nil, nil
			}
			if failure != "Helm factory" {
				var err error
				if failure == "API error" {
					err = errors.New("creation denied")
				}
				f.api.EXPECT().CreateDataplane(gomock.Any(), gomock.Any()).Return(nil, err)
			}
			require.Error(t, cmd.installComponents(context.Background(), f.store, f.api, factory, nil, f.ui, f.context, "install", "region"))
		})
	}
}

func TestInstallComponentsMoveRetainsOldClusterForCleanup(t *testing.T) {
	f := componentTestFixture(t, savedComponentInstallation())
	f.api.EXPECT().RegisterComponent(gomock.Any(), api.RegisterComponentRequest{DataplaneID: "dp", Component: "agui", URL: "https://agui.example.com"}).Return(&api.ComponentCredentials{}, nil)
	cmd := &DataplaneCmd{Namespace: "airbyte", ChartVersion: "2.1.0", Values: []string{componentValuesFile(t, `components: {aguiFrontend: {enabled: true, cluster: secondary, publicIngress: {host: agui.example.com}}}`)}, ClusterContext: map[string]string{"secondary": "ctx-b"}}
	require.NoError(t, cmd.installComponents(context.Background(), f.store, f.api, f.helmFactory(t, false), nil, f.ui, f.context, "install", "unused"))
	require.Len(t, f.values, 2)
	require.Equal(t, "primary", f.values["ctx-a"]["targetCluster"])
	require.Equal(t, "secondary", f.values["ctx-b"]["targetCluster"])
	require.Len(t, f.persisted(t).Clusters, 2)
}

func TestInstallComponentsFailures(t *testing.T) {
	for _, failure := range []string{"helm", "register", "configure", "missing credentials", "client factory", "unregister"} {
		t.Run(failure, func(t *testing.T) {
			state := savedComponentInstallation()
			if failure == "unregister" {
				state.Registered["aguiFrontend"] = "agui"
			}
			f := componentTestFixture(t, state)
			contents := `components: {cloudMcp: {enabled: true, cluster: primary, publicIngress: {host: mcp.example.com}}}`
			if failure == "unregister" {
				contents = `components: {}`
			}
			cmd := &DataplaneCmd{Namespace: "airbyte", ChartVersion: "2.1.0", Values: []string{componentValuesFile(t, contents)}}
			if failure == "unregister" {
				f.api.EXPECT().UnregisterComponent(gomock.Any(), gomock.Any()).Return(errors.New("unregister blocked"))
			} else if failure != "helm" {
				credentials := &api.ComponentCredentials{ClientID: "client", ClientSecret: "test-only"}
				var err error
				if failure == "register" {
					err = errors.New("register blocked")
				}
				if failure == "missing credentials" {
					credentials = &api.ComponentCredentials{}
				}
				f.api.EXPECT().RegisterComponent(gomock.Any(), gomock.Any()).Return(credentials, err)
			}
			cmd.componentClientFactory = func(string, string) (k8s.ComponentClient, error) {
				if failure == "client factory" {
					return nil, errors.New("client blocked")
				}
				return cloudConfigFunc(func(context.Context, string, string, string, string) error { return errors.New("configure blocked") }), nil
			}
			err := cmd.installComponents(context.Background(), f.store, f.api, f.helmFactory(t, failure == "helm"), nil, f.ui, f.context, "install", "region")
			require.Error(t, err)
			if failure == "unregister" {
				require.Equal(t, "agui", f.persisted(t).Registered["aguiFrontend"])
			}
			if failure == "configure" || failure == "register" {
				require.Equal(t, "cloud_mcp", f.persisted(t).Registered["cloudMcp"])
			}
		})
	}
}

func TestInstallComponentValidationBeforeSideEffects(t *testing.T) {
	for _, tc := range []struct {
		name, contents string
		mapping        map[string]string
		version        string
	}{
		{"missing mapping", `components: {x: {enabled: true, cluster: a}, y: {enabled: true, cluster: b}}`, nil, "2.1.0"},
		{"duplicate context", `components: {x: {enabled: true, cluster: a}, y: {enabled: true, cluster: b}}`, map[string]string{"a": "same", "b": "same"}, "2.1.0"},
		{"empty context", `components: {}`, map[string]string{"a": ""}, "2.1.0"},
		{"v1", `components: {}`, nil, "1.8.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := componentTestFixture(t, nil)
			cmd := &DataplaneCmd{Namespace: "airbyte", ChartVersion: tc.version, Values: []string{componentValuesFile(t, tc.contents)}, ClusterContext: tc.mapping}
			factory := func(string, string, string) (goHelm.Client, error) {
				t.Fatal("Helm factory called before validation")
				return nil, nil
			}
			require.Error(t, cmd.installComponents(context.Background(), f.store, f.api, factory, nil, f.ui, f.context, "install", "region"))
		})
	}
	for _, change := range []string{"namespace", "namespace with mapping", "context"} {
		t.Run(change, func(t *testing.T) {
			f := componentTestFixture(t, savedComponentInstallation())
			cmd := &DataplaneCmd{Namespace: "airbyte", ChartVersion: "2.1.0"}
			if strings.HasPrefix(change, "namespace") {
				cmd.Namespace = "different"
				if change == "namespace with mapping" {
					cmd.ClusterContext = map[string]string{"primary": "ctx-a"}
				}
			} else {
				cmd.ClusterContext = map[string]string{"primary": "different"}
			}
			require.Error(t, cmd.installComponents(context.Background(), f.store, f.api, nil, nil, f.ui, f.context, "install", "region"))
		})
	}
}

func TestInstallationCheckpointFailures(t *testing.T) {
	store := airboxmock.NewMockConfigStore(gomock.NewController(t))
	store.EXPECT().Load().Return(nil, errors.New("read denied"))
	require.ErrorContains(t, saveInstallation(store, "install", savedComponentInstallation()), "cannot be loaded")
	store.EXPECT().Load().Return(&airbox.Config{CurrentContext: "cloud", Contexts: []airbox.NamedContext{{Name: "cloud", Context: airbox.Context{OrganizationID: "org"}}}}, nil)
	store.EXPECT().Save(gomock.Any()).Return(errors.New("write denied"))
	require.ErrorContains(t, saveInstallation(store, "install", savedComponentInstallation()), "cannot be saved")
	require.NotEqual(t, installationKey("org-a", "install"), installationKey("org-b", "install"))
}
