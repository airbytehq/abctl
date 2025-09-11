package install

import (
	"context"
	"fmt"
	"sort"

	goHelm "github.com/mittwald/go-helm-client"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/api"
	"github.com/airbytehq/abctl/internal/helm"
	"github.com/airbytehq/abctl/internal/http"
	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/ui"
)

// DataplaneCmd handles dataplane installation
type DataplaneCmd struct {
	Namespace string `short:"n" help:"Target namespace (default: 'default' for Kind clusters, current kubeconfig context for existing clusters)."`
	// Hide flag for now.
	WithKindCluster bool `default:"true" help:"Create a Kind cluster for the dataplane and set it as current context." hidden:""`
}

// Run executes the install dataplane command
func (c *DataplaneCmd) Run(
	ctx context.Context,
	cfg airbox.ConfigStore,
	httpClient http.HTTPDoer,
	apiFactory airbox.APIServiceFactory,
	helmFactory helm.Factory,
	clusterFactory k8s.ClusterFactory,
	ui ui.Provider,
) error {
	ui.Title("Starting interactive dataplane installation")

	// Setup namespace
	if err := c.resolveNamespace(); err != nil {
		return err
	}

	// Initialize API client
	apiClient, err := apiFactory(ctx, httpClient, cfg)
	if err != nil {
		return err
	}

	// Validate config has org set
	abContext, err := loadContext(cfg)
	if err != nil {
		return err
	}

	// Get region and dataplane name from user
	region, err := selectOrCreateRegion(ctx, ui, apiClient, abContext.OrganizationID)
	if err != nil {
		return err
	}

	name, err := getDataplaneName(ui)
	if err != nil {
		return err
	}

	// Setup Kind cluster if needed
	var kubeconfig, kubeContext, clusterName string
	if c.WithKindCluster {
		kubeconfig, kubeContext, clusterName, err = c.setupKindCluster(ctx, ui, clusterFactory, name)
		if err != nil {
			return err
		}
	}

	// Create Helm client
	helmClient, err := helmFactory(kubeconfig, kubeContext, c.Namespace)
	if err != nil {
		return fmt.Errorf("failed to create helm client: %w", err)
	}

	// Register dataplane with API
	creds, err := registerDataplane(ctx, ui, apiClient, name, region.ID, abContext.OrganizationID)
	if err != nil {
		return err
	}

	// Deploy to Kubernetes
	if err := deployChart(ctx, ui, helmClient, c.Namespace, name, creds, abContext); err != nil {
		return err
	}

	// Show success
	c.showSuccess(ui, name, clusterName)
	return nil
}

// resolveNamespace sets the namespace based on cluster type
func (c *DataplaneCmd) resolveNamespace() error {
	if c.Namespace != "" {
		return nil
	}

	if c.WithKindCluster {
		c.Namespace = "default"
		return nil
	}

	ns, err := k8s.GetCurrentNamespace()
	if err != nil {
		return fmt.Errorf("failed to get namespace from current context: %w", err)
	}
	c.Namespace = ns
	return nil
}

// loadContext validates and returns the current context
func loadContext(cfg airbox.ConfigStore) (*airbox.Context, error) {
	config, err := cfg.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	abContext, err := config.GetCurrentContext()
	if err != nil {
		return nil, fmt.Errorf("failed to get current context: %w", err)
	}

	// Make sure the organization id is set. Catch this early
	// rather than relying on the create region API response.
	if abContext.OrganizationID == "" {
		return nil, airbox.NewLoginError("no organization set in context")
	}

	return abContext, nil
}

// setupKindCluster creates a Kind cluster and returns connection details
func (c *DataplaneCmd) setupKindCluster(ctx context.Context, ui ui.Provider, factory k8s.ClusterFactory, dataplaneName string) (kubeconfig, kubeContext, clusterName string, err error) {
	clusterName = fmt.Sprintf("airbox-%s", dataplaneName)

	cluster, err := factory(ctx, clusterName)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create cluster: %w", err)
	}

	err = ui.RunWithSpinner("Creating Kind cluster", func() error {
		return cluster.Create(ctx, 0, nil)
	})
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create Kind cluster: %w", err)
	}

	kubeconfig = k8s.DefaultProvider.Kubeconfig
	kubeContext = fmt.Sprintf("kind-%s", clusterName)
	return kubeconfig, kubeContext, clusterName, nil
}

// registerDataplane creates the dataplane via API
func registerDataplane(ctx context.Context, ui ui.Provider, apiClient api.Service, name, regionID, orgID string) (*api.CreateDataplaneResponse, error) {
	var response *api.CreateDataplaneResponse

	err := ui.RunWithSpinner("Creating dataplane", func() error {
		var err error
		response, err = apiClient.CreateDataplane(ctx, api.CreateDataplaneRequest{
			Name:           name,
			RegionID:       regionID,
			OrganizationID: orgID,
			Enabled:        true,
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create dataplane: %w", err)
	}

	ui.ShowSection("Dataplane Credentials:",
		"DataplaneID: "+response.DataplaneID,
		"ClientID: "+response.ClientID,
		"ClientSecret: "+response.ClientSecret,
	)

	return response, nil
}

// deployChart installs the Helm chart
func deployChart(ctx context.Context, ui ui.Provider, client goHelm.Client, namespace, name string, creds *api.CreateDataplaneResponse, context *airbox.Context) error {
	return ui.RunWithSpinner("Installing dataplane chart", func() error {
		return helm.InstallDataplaneChart(ctx, client, namespace, name, creds, context)
	})
}

// showSuccess displays success message and kubectl instructions
func (c *DataplaneCmd) showSuccess(ui ui.Provider, name, clusterName string) {
	ui.ShowSuccess(fmt.Sprintf("Dataplane '%s' installed successfully!", name))
	ui.NewLine()

	if c.WithKindCluster && clusterName != "" {
		ui.ShowHeading("To use kubectl with this Kind cluster")
		ui.ShowKeyValue("1. Export kubeconfig", fmt.Sprintf("kind export kubeconfig --name %s", clusterName))
		if c.Namespace != "default" {
			ui.ShowKeyValue("2. Set namespace", fmt.Sprintf("kubectl config set-context --current --namespace=%s", c.Namespace))
		}
		ui.NewLine()
	}
}

// selectOrCreateRegion handles the region selection or creation flow
func selectOrCreateRegion(ctx context.Context, ui ui.Provider, apiClient api.Service, organizationID string) (*api.Region, error) {
	regionTypeOptions := []string{"Use existing region", "Create new region"}
	regionTypeIndex, _, err := ui.Select("Select region option:", regionTypeOptions)
	if err != nil {
		return nil, fmt.Errorf("region type selection cancelled: %w", err)
	}

	if regionTypeIndex == 0 {
		return selectExistingRegion(ctx, ui, apiClient, organizationID)
	}
	return createNewRegion(ctx, ui, apiClient, organizationID)
}

// selectExistingRegion handles selection from existing regions
func selectExistingRegion(ctx context.Context, ui ui.Provider, apiClient api.Service, organizationID string) (*api.Region, error) {
	regions, err := apiClient.ListRegions(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch regions: %w", err)
	}

	if len(regions) == 0 {
		return nil, fmt.Errorf("no regions available")
	}

	// Sort regions by name for better UX
	sort.Slice(regions, func(i, j int) bool {
		return regions[i].Name < regions[j].Name
	})

	regionOptions := make([]string, len(regions))
	for i, region := range regions {
		display := region.Name
		if region.Location != "" {
			display = fmt.Sprintf("%s - %s", region.Name, region.Location)
		}
		if region.CloudProvider != "" {
			display = fmt.Sprintf("%s (%s)", display, region.CloudProvider)
		}
		regionOptions[i] = display
	}

	regionIndex, _, err := ui.FilterableSelect("Select existing region:", regionOptions)
	if err != nil {
		return nil, fmt.Errorf("region selection cancelled: %w", err)
	}
	return regions[regionIndex], nil
}

// createNewRegion handles creating a new region
func createNewRegion(ctx context.Context, ui ui.Provider, apiClient api.Service, organizationID string) (*api.Region, error) {
	regionName, err := ui.TextInput("Enter new region name:", "my-region", nil)
	if err != nil {
		return nil, fmt.Errorf("region name input cancelled: %w", err)
	}

	createRegionRequest := api.CreateRegionRequest{
		Name:           regionName,
		OrganizationID: organizationID,
	}

	newRegion, err := apiClient.CreateRegion(ctx, createRegionRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create region: %w", err)
	}

	return newRegion, nil
}

// getDataplaneName prompts for and validates the dataplane name
func getDataplaneName(ui ui.Provider) (string, error) {
	name, err := ui.TextInput("Enter dataplane name:", "my-dataplane", airbox.ValidateDataplaneName)
	if err != nil {
		return "", fmt.Errorf("dataplane name input cancelled: %w", err)
	}
	return name, nil
}
