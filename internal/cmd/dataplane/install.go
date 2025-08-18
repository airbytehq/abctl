package dataplane

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/airbytehq/abctl/internal/auth/oidc"
	"github.com/airbytehq/abctl/internal/helm"
	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/service"
	"github.com/pterm/pterm"
)

type InstallCmd struct {
	ClientID     string `flag:"" help:"Dataplane client ID (overrides credentials file)"`
	ClientSecret string `flag:"" help:"Dataplane client secret (overrides credentials file)"`
	Values       string `flag:"" help:"Path to custom Helm values file" type:"path"`
	Chart        string `flag:"" help:"Path to Helm chart (defaults to embedded chart)" type:"path"`
	ChartVersion string `flag:"" help:"Version of the Helm chart to install"`
	Namespace    string `flag:"" help:"Kubernetes namespace to install into" default:"airbyte-dataplane"`
	Name         string `flag:"" help:"Helm release name" default:"airbyte-dataplane"`
}

func (c *InstallCmd) Run(ctx context.Context, provider k8s.Provider, managerFactory service.ManagerClientFactory) error {
	pterm.Info.Println("Installing Airbyte with dataplane configuration...")
	
	// Ensure we're using the dataplane cluster name and context
	if provider.ClusterName != "airbyte-dataplane" {
		pterm.Info.Printf("Updating cluster name to 'airbyte-dataplane' (was: %s)\n", provider.ClusterName)
		provider.ClusterName = "airbyte-dataplane"
		provider.Context = "kind-airbyte-dataplane"
	}
	
	// Check if dataplane cluster exists, create if needed
	cluster, err := provider.Cluster(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize kind cluster: %w", err)
	}
	
	if !cluster.Exists(ctx) {
		pterm.Info.Printf("Creating dataplane cluster '%s'...\n", provider.ClusterName)
		if err := cluster.Create(ctx, 0, nil); err != nil { // No port mapping needed for dataplane
			return fmt.Errorf("failed to create dataplane cluster: %w", err)
		}
		pterm.Success.Printf("Dataplane cluster '%s' created successfully\n", provider.ClusterName)
	} else {
		pterm.Info.Printf("Using existing dataplane cluster '%s'\n", provider.ClusterName)
	}
	
	// Load dataplane credentials
	var clientID, clientSecret, airbyteURL string
	var dataplaneInfo *oidc.DataPlaneInfo
	
	// Check if flags were provided
	if c.ClientID != "" && c.ClientSecret != "" {
		clientID = c.ClientID
		clientSecret = c.ClientSecret
		pterm.Info.Printf("Using dataplane credentials from flags\n")
	} else {
		// Try to load from credentials file
		pterm.Info.Println("Loading dataplane credentials from ~/.abctl/credentials...")
		creds, err := oidc.LoadCredentials()
		if err != nil {
			return fmt.Errorf("failed to load credentials: %w\nPlease run 'abctl dataplane create' first or provide --client-id and --client-secret flags", err)
		}
		
		if creds.DataPlane == nil {
			return fmt.Errorf("no dataplane credentials found in ~/.abctl/credentials\nPlease run 'abctl dataplane create' first or provide --client-id and --client-secret flags")
		}
		
		dataplaneInfo = creds.DataPlane
		clientID = dataplaneInfo.ClientID
		clientSecret = dataplaneInfo.ClientSecret
		airbyteURL = creds.BaseURL
		
		pterm.Info.Printf("Using dataplane: %s (ID: %s)\n", dataplaneInfo.Name, dataplaneInfo.DataPlaneID)
		pterm.Info.Printf("Region: %s\n", dataplaneInfo.RegionID)
		pterm.Info.Printf("Organization: %s\n", dataplaneInfo.OrganizationID)
		pterm.Info.Printf("Airbyte URL: %s\n", airbyteURL)
	}
	
	// Validate credentials
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("client ID and client secret are required")
	}
	
	// Mask the client secret for display
	maskedSecret := maskSecret(clientSecret)
	pterm.Info.Printf("Client ID: %s\n", clientID)
	pterm.Info.Printf("Client Secret: %s\n", maskedSecret)
	
	// Load the required service manager clients with dataplane namespace
	k8sClient, err := service.DefaultK8s(provider.Kubeconfig, provider.Context)
	if err != nil {
		return fmt.Errorf("failed to initialize the kubernetes client: %w", err)
	}

	helmClient, err := helm.New(provider.Kubeconfig, provider.Context, c.Namespace)
	if err != nil {
		return fmt.Errorf("failed to initialize the helm client: %w", err)
	}
	
	// Determine and set defaults for chart flags
	resolver := helm.NewChartResolver(helmClient)
	resolvedChart, resolvedVersion, err := resolver.ResolveChartReference(c.Chart, c.ChartVersion)
	if err != nil {
		return fmt.Errorf("failed to resolve chart flags: %w", err)
	}
	c.Chart = resolvedChart
	c.ChartVersion = resolvedVersion
	
	// Create service manager
	manager, err := service.NewManager(provider,
		service.WithK8sClient(k8sClient),
		service.WithHelmClient(helmClient),
	)
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	// Create namespace if it doesn't exist
	if !k8sClient.NamespaceExists(ctx, c.Namespace) {
		pterm.Info.Printf("Creating namespace '%s'\n", c.Namespace)
		if err := k8sClient.NamespaceCreate(ctx, c.Namespace); err != nil {
			return fmt.Errorf("unable to create namespace '%s': %w", c.Namespace, err)
		}
		pterm.Info.Printf("Namespace '%s' created\n", c.Namespace)
	} else {
		pterm.Info.Printf("Namespace '%s' already exists\n", c.Namespace)
	}

	// Create the airbyte-auth-secrets secret for dataplane credentials
	pterm.Info.Println("Creating dataplane auth secrets...")
	if err := manager.HandleAuthSecret(ctx, c.Namespace, clientID, clientSecret); err != nil {
		return fmt.Errorf("failed to create auth secrets: %w", err)
	}
	
	// Build the values.yaml file for the Airbyte chart
	valuesOpts := helm.ValuesOpts{
		ValuesFile: c.Values,
	}
	
	valuesYAML, err := helm.BuildAirbyteValues(ctx, valuesOpts, c.ChartVersion)
	if err != nil {
		return fmt.Errorf("failed to build values yaml: %w", err)
	}

	// Add dataplane-specific values to disable unnecessary components
	dataplaneValues := `
airbyteBootloader:
  enabled: false
connectorBuilderServer:
  enabled: false
workloadApiServer:
  enabled: false
`
	valuesYAML += dataplaneValues

	// Prepare installation options
	installOpts := &service.InstallOpts{
		AirbyteChartLoc:  c.Chart,
		HelmChartVersion: c.ChartVersion,
		HelmValuesYaml:   valuesYAML,
		Namespace:        c.Namespace,
		DataPlane: &service.DataPlaneConfig{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			AirbyteURL:   airbyteURL,
		},
	}
	
	// Add dataplane info if available
	if dataplaneInfo != nil {
		installOpts.DataPlane.DataPlaneID = dataplaneInfo.DataPlaneID
		installOpts.DataPlane.RegionID = dataplaneInfo.RegionID
		installOpts.DataPlane.OrganizationID = dataplaneInfo.OrganizationID
	}
	
	pterm.Info.Println("Starting Airbyte installation with dataplane configuration...")
	
	// Install Airbyte with dataplane configuration
	if err := manager.Install(ctx, installOpts); err != nil {
		return fmt.Errorf("failed to install Airbyte: %w", err)
	}
	
	pterm.Success.Println("✓ Airbyte installed successfully with dataplane configuration!")
	
	// Display post-installation information
	pterm.Info.Println("\nAirbyte dataplane is now running.")
	pterm.Info.Println("The workloads will connect to your configured dataplane for data synchronization.")
	if airbyteURL != "" {
		pterm.Info.Printf("Connected to Airbyte instance: %s\n", airbyteURL)
	}
	
	return nil
}

// maskSecret masks all but the first 4 characters of a secret
func maskSecret(secret string) string {
	if len(secret) <= 4 {
		return "****"
	}
	return secret[:4] + strings.Repeat("*", len(secret)-4)
}

// loadValuesFile reads the content of a values file
func loadValuesFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read values file %s: %w", path, err)
	}
	return string(content), nil
}

