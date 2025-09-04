package install

import (
	"context"
	"fmt"

	"github.com/airbytehq/abctl/internal/abctl"
	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/api"
	"github.com/airbytehq/abctl/internal/helm"
	"github.com/airbytehq/abctl/internal/http"
	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/service"
	"github.com/airbytehq/abctl/internal/ui"
	goHelm "github.com/mittwald/go-helm-client"
)

// DataplaneCmd handles dataplane installation
type DataplaneCmd struct {
	Namespace string `short:"n" help:"Target namespace (default: current kubeconfig context)." hidden:""`
	// Hide flag for now.
	WithKindCluster bool `default:"true" help:"Create a Kind cluster for the dataplane and set it as current context." hidden:""`
}

// Run executes the install dataplane command
func (c *DataplaneCmd) Run(ctx context.Context, httpClient http.HTTPDoer, apiFactory api.Factory, helmFactory helm.Factory, cfg airbox.ConfigProvider, ui ui.Provider) error {
	ui.Title("Starting interactive dataplane installation")

	// Resolve initial namespace for API operations
	if c.Namespace == "" {
		if c.WithKindCluster {
			// When creating a Kind cluster, use default namespace
			c.Namespace = "default"
		} else {
			// For existing clusters, resolve from current context
			var err error
			c.Namespace, err = k8s.GetCurrentNamespace()
			if err != nil {
				return fmt.Errorf("failed to get namespace from current context: %w", err)
			}
		}
	}

	// Get organization ID from local airbox config
	abCfg, err := cfg.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := abCfg.IsAuthenticated(); err != nil {
		return err
	}

	currentContext, err := abCfg.GetCurrentContext()
	if err != nil {
		return fmt.Errorf("failed to get current context: %w", err)
	}

	// Check organization context
	if currentContext.OrganizationID == "" {
		return fmt.Errorf("no organization context found. Please run 'airbox auth login' to set workspace")
	}

	// Use organization ID from current context
	orgID := currentContext.OrganizationID

	// Create API client using factory (works for both modes)
	apiClient, err := apiFactory(ctx, cfg, httpClient)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Create helm client if not using Kind cluster
	var helmClient goHelm.Client
	if !c.WithKindCluster {
		helmClient, err = helmFactory("", "", c.Namespace)
		if err != nil {
			return fmt.Errorf("failed to create helm client: %w", err)
		}
	}

	// Step 2: Select New or Existing Region
	regionTypeOptions := []string{"Use existing region", "Create new region"}
	regionTypeIndex, _, err := ui.Select("Select region option:", regionTypeOptions)
	if err != nil {
		return fmt.Errorf("region type selection cancelled: %w", err)
	}

	var selectedRegion *api.Region

	if regionTypeIndex == 0 {
		// Use existing region
		regions, err := apiClient.ListRegions(ctx, orgID)
		if err != nil {
			return fmt.Errorf("failed to fetch regions: %w", err)
		}

		if len(regions) == 0 {
			return fmt.Errorf("no regions available")
		}

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

		regionIndex, _, err := ui.Select("Select existing region:", regionOptions)
		if err != nil {
			return fmt.Errorf("region selection cancelled: %w", err)
		}
		selectedRegion = regions[regionIndex]
	} else {
		// Create new region
		regionName, err := ui.TextInput("Enter new region name:", "my-region", nil)
		if err != nil {
			return fmt.Errorf("region name input cancelled: %w", err)
		}

		// Create the region via API
		createRegionRequest := api.CreateRegionRequest{
			Name:           regionName,
			OrganizationID: orgID,
		}

		newRegion, err := apiClient.CreateRegion(ctx, createRegionRequest)
		if err != nil {
			return fmt.Errorf("failed to create region: %w", err)
		}

		selectedRegion = newRegion
	}

	// Step 3: Get Dataplane Name
	dataplaneName, err := ui.TextInput("Enter dataplane name:", "my-dataplane", validateDataplaneName)
	if err != nil {
		return fmt.Errorf("dataplane name input cancelled: %w", err)
	}

	// Step 3.5: Create Kind cluster if requested
	var kindKubeconfig, kindContext, kindNamespace string
	kindClusterName := fmt.Sprintf("airbox-%s", dataplaneName)
	if c.WithKindCluster {
		err := ui.RunWithSpinner(
			fmt.Sprintf("Creating Kind cluster '%s'", kindClusterName),
			func() error {
				kindKubeconfig, kindContext, kindNamespace, err = c.createKindCluster(ctx, kindClusterName)
				return err
			},
		)
		if err != nil {
			return fmt.Errorf("failed to create Kind cluster: %w", err)
		}

		// Reload k8s and helm clients to point to Kind cluster using direct calls
		// (API client stays unchanged - must continue talking to control plane)
		// k8s client not needed for this operation - only helm client for chart installation
		_, err = service.DefaultK8s(kindKubeconfig, kindContext)
		if err != nil {
			return fmt.Errorf("failed to create Kind k8s client: %w", err)
		}

		helmClient, err = helm.New(kindKubeconfig, kindContext, kindNamespace)
		if err != nil {
			return fmt.Errorf("failed to create Kind helm client: %w", err)
		}
	}

	// Step 4: Create Dataplane via API
	var createResponse *api.CreateDataplaneResponse
	err = ui.RunWithSpinner("Creating dataplane", func() error {
		var err error
		createResponse, err = apiClient.CreateDataplane(ctx, api.CreateDataplaneRequest{
			Name:           dataplaneName,
			RegionID:       selectedRegion.ID,
			OrganizationID: orgID,
			Enabled:        true,
		})
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to create dataplane: %w", err)
	}

	ui.ShowSection("Dataplane Credentials:",
		"DataplaneID: "+createResponse.DataplaneID,
		"ClientID: "+createResponse.ClientID,
		"ClientSecret: "+createResponse.ClientSecret,
	)

	// Step 5: Install Airbyte dataplane Helm chart
	// helmClient already created by factory

	// Convert airbox context to abctl config format for helm chart
	abctlConfig := &abctl.Config{
		AirbyteAPIHost: currentContext.AirbyteAPIHost,
		AirbyteURL:     currentContext.AirbyteURL,
		AirbyteAuthURL: currentContext.AirbyteAuthURL,
		OIDCClientID:   currentContext.OIDCClientID,
		Edition:        currentContext.Edition,
		OrganizationID: currentContext.OrganizationID,
	}

	// Install dataplane chart with spinner
	err = ui.RunWithSpinner("Installing dataplane chart", func() error {
		return helm.InstallDataplaneChart(ctx, helmClient, c.Namespace, dataplaneName, createResponse, abctlConfig)
	})
	if err != nil {
		return fmt.Errorf("failed to install dataplane chart: %w", err)
	}

	ui.ShowSuccess(fmt.Sprintf("Dataplane '%s' installed successfully!", dataplaneName))
	ui.NewLine()

	// Show Kind cluster context instructions if cluster was created
	if c.WithKindCluster {
		ui.ShowHeading("To use kubectl with this Kind cluster")
		ui.ShowKeyValue("Export kubeconfig", fmt.Sprintf("kind export kubeconfig --name %s", kindClusterName))
		ui.NewLine()
	}

	return nil
}

func validateDataplaneName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if len(name) > 63 {
		return fmt.Errorf("name cannot exceed 63 characters")
	}

	// Must start with a letter
	if name[0] < 'a' || name[0] > 'z' {
		return fmt.Errorf("name must start with a lowercase letter")
	}

	// Only lowercase alphanumeric and hyphens allowed
	for i, char := range name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '-') {
			return fmt.Errorf("name can only contain lowercase letters, numbers, and hyphens (invalid character at position %d)", i+1)
		}
	}

	// Cannot end with hyphen
	if name[len(name)-1] == '-' {
		return fmt.Errorf("name cannot end with a hyphen")
	}

	return nil
}

// createKindCluster creates a Kind cluster and returns connection details
func (c *DataplaneCmd) createKindCluster(ctx context.Context, clusterName string) (kubeconfig, context, namespace string, err error) {
	// Create a Kind provider for the new cluster
	provider := k8s.Provider{
		Name:        k8s.Kind,
		ClusterName: clusterName,
		Context:     fmt.Sprintf("kind-%s", clusterName),
		Kubeconfig:  k8s.DefaultProvider.Kubeconfig,
	}

	// Get the cluster interface
	cluster, err := provider.Cluster(ctx)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create cluster interface: %w", err)
	}

	// Create the Kind cluster
	if err := cluster.Create(ctx, 0, nil); err != nil {
		return "", "", "", fmt.Errorf("failed to create cluster: %w", err)
	}

	// Return cluster connection details
	return provider.Kubeconfig, provider.Context, "default", nil
}
