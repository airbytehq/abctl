package install

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/api"
	"github.com/airbytehq/abctl/internal/helm"
	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/ui"
	goHelm "github.com/mittwald/go-helm-client"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/tools/clientcmd"
)

var registeredComponents = map[string]string{
	"cloudMcp":     "cloud_mcp",
	"mtxApi":       "mtx_api",
	"aguiFrontend": "agui",
}

type componentSpec struct {
	Enabled       bool   `yaml:"enabled"`
	Cluster       string `yaml:"cluster"`
	PublicIngress struct {
		Host string `yaml:"host"`
	} `yaml:"publicIngress"`
}

func loadComponentValues(files []string) (map[string]any, map[string]componentSpec, error) {
	values := map[string]any{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, nil, fmt.Errorf("values overlay cannot be read: %w", err)
		}
		var overlay map[string]any
		if err := yaml.Unmarshal(data, &overlay); err != nil {
			return nil, nil, fmt.Errorf("values overlay is not a YAML mapping. Check the --values files.")
		}
		values = helm.MergeValues(values, overlay)
	}
	data, err := yaml.Marshal(values["components"])
	if err != nil {
		return nil, nil, fmt.Errorf("components values cannot be encoded")
	}
	components := map[string]componentSpec{}
	if err := yaml.Unmarshal(data, &components); err != nil {
		return nil, nil, fmt.Errorf("components must be a map with boolean enabled and string cluster/host fields")
	}
	for key, component := range components {
		if !component.Enabled {
			continue
		}
		if component.Cluster == "" {
			return nil, nil, fmt.Errorf("components.%s.cluster is required. Set a cluster name in the values overlay.", key)
		}
		if component.PublicIngress.Host != "" && len(validation.IsDNS1123Subdomain(component.PublicIngress.Host)) > 0 {
			return nil, nil, fmt.Errorf("components.%s.publicIngress.host must be a DNS hostname without a scheme or path", key)
		}
	}
	return values, components, nil
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func installationKey(organizationID, name string) string {
	return organizationID + "/" + name
}

func saveInstallation(cfg airbox.ConfigStore, name string, installation *airbox.DataplaneInstallation) error {
	config, err := cfg.Load()
	if err != nil {
		return fmt.Errorf("installation configuration cannot be loaded: %w", err)
	}
	current, err := config.GetCurrentContext()
	if err != nil {
		return err
	}
	if current.DataplaneInstallations == nil {
		current.DataplaneInstallations = map[string]*airbox.DataplaneInstallation{}
	}
	current.DataplaneInstallations[installationKey(current.OrganizationID, name)] = installation
	config.AddContext(config.CurrentContext, *current)
	if err := cfg.Save(config); err != nil {
		return fmt.Errorf("installation state cannot be saved. Restore write access to AIRBOXCONFIG and retry: %w", err)
	}
	return nil
}

func (c *DataplaneCmd) installComponents(ctx context.Context, cfg airbox.ConfigStore, apiClient api.Service, helmFactory helm.Factory, clusterFactory k8s.ClusterFactory, output ui.Provider, abContext *airbox.Context, name, regionID string) error {
	values, components, err := loadComponentValues(c.Values)
	if err != nil {
		return err
	}
	if c.ChartVersion != "" && !helm.ChartIsV2PlusBaseVersion(c.ChartVersion) {
		return fmt.Errorf("components require chart V2. Select --chart-version 2.0.0 or newer.")
	}
	installation := abContext.DataplaneInstallations[installationKey(abContext.OrganizationID, name)]
	targets := map[string]airbox.ComponentCluster{}
	if installation != nil {
		for key, target := range installation.Clusters {
			targets[key] = target
		}
	}
	for _, key := range sortedKeys(components) {
		component := components[key]
		if component.Enabled {
			if _, exists := targets[component.Cluster]; !exists {
				targets[component.Cluster] = airbox.ComponentCluster{Namespace: c.Namespace}
			}
			if _, registered := registeredComponents[key]; !registered {
				output.ShowInfo(fmt.Sprintf("Component %s is install-only; control-plane registration skipped.", key))
			} else if component.PublicIngress.Host == "" {
				output.ShowInfo(fmt.Sprintf("Component %s has no publicIngress.host; control-plane registration skipped.", key))
			}
		}
	}
	if len(targets) == 0 {
		cluster, _ := values["targetCluster"].(string)
		if cluster == "" {
			cluster = "primary"
			if len(c.ClusterContext) == 1 {
				cluster = sortedKeys(c.ClusterContext)[0]
			}
		}
		targets[cluster] = airbox.ComponentCluster{Namespace: c.Namespace}
	}
	for cluster, kubeContext := range c.ClusterContext {
		if cluster == "" || kubeContext == "" {
			return fmt.Errorf("--cluster-context requires nonempty cluster=context pairs")
		}
		previous, exists := targets[cluster]
		if !exists {
			continue
		}
		if previous.Namespace != c.Namespace {
			return fmt.Errorf("the saved release namespace differs. Use the original --namespace to upgrade this installation.")
		}
		if previous.Context != "" && previous.Context != kubeContext {
			return fmt.Errorf("a saved cluster context cannot be replaced. Uninstall the old release before changing --cluster-context.")
		}
		previous.Context, previous.Namespace = kubeContext, c.Namespace
		targets[cluster] = previous
	}
	for _, target := range targets {
		if target.Namespace != c.Namespace {
			return fmt.Errorf("the saved release namespace differs. Use the original --namespace to upgrade this installation.")
		}
		if target.Context == "" && (len(targets) > 1 || len(c.ClusterContext) > 0) {
			return fmt.Errorf("each component cluster requires a kubeconfig context. Supply --cluster-context cluster=context for every cluster.")
		}
	}
	seen := map[string]bool{}
	for _, cluster := range sortedKeys(targets) {
		target := targets[cluster]
		if target.Context == "" {
			if c.WithKindCluster {
				config, kubeContext, _, err := c.setupKindCluster(ctx, output, clusterFactory, name)
				if err != nil {
					return err
				}
				target.Kubeconfig, target.Context = config, kubeContext
			} else {
				config, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
				if err != nil || config.CurrentContext == "" {
					return fmt.Errorf("the current kubeconfig context is unavailable. Supply --cluster-context cluster=context.")
				}
				target.Context = config.CurrentContext
			}
			targets[cluster] = target
		}
		if seen[target.Context] {
			return fmt.Errorf("component cluster names must resolve to different kubeconfig contexts")
		}
		seen[target.Context] = true
	}
	chartURL, version, repoURL, err := helm.ResolveDataplaneChartReference(c.ChartVersion)
	if err != nil {
		return fmt.Errorf("dataplane chart resolution failed: %w", err)
	}
	clients := map[string]goHelm.Client{}
	for _, cluster := range sortedKeys(targets) {
		target := targets[cluster]
		client, err := helmFactory(target.Kubeconfig, target.Context, target.Namespace)
		if err != nil {
			return fmt.Errorf("Helm client creation for component cluster failed: %w", err)
		}
		clients[cluster] = client
	}
	if installation == nil {
		credentials, err := registerDataplane(ctx, output, apiClient, name, regionID, abContext.OrganizationID)
		if err != nil {
			return err
		}
		if credentials == nil || credentials.DataplaneID == "" || credentials.ClientID == "" || credentials.ClientSecret == "" {
			return fmt.Errorf("dataplane registration returned incomplete credentials. Retry the installation.")
		}
		installation = &airbox.DataplaneInstallation{DataplaneID: credentials.DataplaneID, ClientID: credentials.ClientID, ClientSecret: credentials.ClientSecret, Registered: map[string]string{}}
	}
	if installation.DataplaneID == "" || installation.ClientID == "" || installation.ClientSecret == "" {
		return fmt.Errorf("saved dataplane credentials are incomplete. Restore this installation's AIRBOXCONFIG credentials before upgrading.")
	}
	installation.Clusters = targets
	if installation.Registered == nil {
		installation.Registered = map[string]string{}
	}
	if err := saveInstallation(cfg, name, installation); err != nil {
		return err
	}
	credentials := &api.CreateDataplaneResponse{DataplaneID: installation.DataplaneID, ClientID: installation.ClientID, ClientSecret: installation.ClientSecret}
	for _, cluster := range sortedKeys(targets) {
		cloud := components["cloudMcp"]
		deferWait := cloud.Enabled && cloud.Cluster == cluster && cloud.PublicIngress.Host != ""
		err := helm.InstallDataplaneChart(ctx, clients[cluster], c.Namespace, name, chartURL, version, repoURL, credentials, abContext, helm.DataplaneChartOptions{Values: values, TargetCluster: cluster, DeferWait: deferWait})
		if err != nil {
			return err
		}
	}
	if err := c.reconcileComponents(ctx, cfg, apiClient, name, installation, values, components); err != nil {
		return err
	}
	c.showSuccess(output, name, "")
	return nil
}

func (c *DataplaneCmd) reconcileComponents(ctx context.Context, cfg airbox.ConfigStore, apiClient api.Service, name string, installation *airbox.DataplaneInstallation, values map[string]any, components map[string]componentSpec) error {
	desired := map[string]string{}
	for key, component := range components {
		if kind, registered := registeredComponents[key]; registered && component.Enabled && component.PublicIngress.Host != "" {
			desired[key] = kind
		}
	}
	for _, key := range sortedKeys(installation.Registered) {
		if _, exists := desired[key]; exists {
			continue
		}
		if err := apiClient.UnregisterComponent(ctx, api.UnregisterComponentRequest{DataplaneID: installation.DataplaneID, Component: installation.Registered[key]}); err != nil {
			return err
		}
		delete(installation.Registered, key)
		if err := saveInstallation(cfg, name, installation); err != nil {
			return err
		}
	}
	for _, key := range sortedKeys(desired) {
		component := components[key]
		// Save intent before the API call so a failed credential write can still be unregistered.
		installation.Registered[key] = desired[key]
		if err := saveInstallation(cfg, name, installation); err != nil {
			return err
		}
		credentials, err := apiClient.RegisterComponent(ctx, api.RegisterComponentRequest{DataplaneID: installation.DataplaneID, Component: desired[key], URL: "https://" + component.PublicIngress.Host})
		if err != nil {
			return err
		}
		if key == "cloudMcp" {
			if credentials == nil || credentials.ClientID == "" || credentials.ClientSecret == "" {
				return fmt.Errorf("Cloud MCP registration returned incomplete credentials. Retry the installation.")
			}
			factory := c.componentClientFactory
			if factory == nil {
				factory = k8s.NewComponentClient
			}
			target := installation.Clusters[component.Cluster]
			client, err := factory(target.Kubeconfig, target.Context)
			if err != nil {
				return err
			}
			namespace := "airbyte-agents"
			if configured, ok := values["componentsNamespace"].(string); ok && configured != "" {
				namespace = configured
			}
			if err := client.ConfigureCloudMCP(ctx, namespace, name, credentials.ClientID, credentials.ClientSecret); err != nil {
				return err
			}
		}
	}
	return nil
}
