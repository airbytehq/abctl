package dataplane

import (
	"context"
	"fmt"
	"strings"

	"github.com/airbytehq/abctl/internal/api"
	"github.com/airbytehq/abctl/internal/auth/oidc"
	"github.com/pterm/pterm"
)

type CreateCmd struct {
	BaseURL    string `flag:"" help:"Base URL of the Airbyte instance" default:"https://cloud.airbyte.com"`
	OIDCServer string `flag:"" help:"OIDC server root URL" default:"https://cloud.airbyte.com/auth"`
}

func (c *CreateCmd) Run() error {
	var baseURL, oidcServer string
	
	// Use flag values if provided, otherwise prompt for input
	if c.BaseURL != "" && c.BaseURL != "https://cloud.airbyte.com" {
		// Flag was explicitly set to a non-default value
		baseURL = c.BaseURL
		pterm.Info.Printf("Using Airbyte instance from flag: %s\n", baseURL)
	} else {
		// Prompt for base URL with default value
		urlPrompt := fmt.Sprintf("Enter the base URL of the Airbyte instance [%s]", c.BaseURL)
		inputURL, _ := pterm.DefaultInteractiveTextInput.Show(urlPrompt)
		
		// Use default if user didn't enter anything
		inputURL = strings.TrimSpace(inputURL)
		if inputURL == "" {
			baseURL = c.BaseURL
		} else {
			baseURL = inputURL
		}
		pterm.Info.Printf("Using Airbyte instance at: %s\n", baseURL)
	}

	if c.OIDCServer != "" && c.OIDCServer != "https://cloud.airbyte.com/auth" {
		// Flag was explicitly set to a non-default value
		oidcServer = c.OIDCServer
		pterm.Info.Printf("Using OIDC server from flag: %s\n", oidcServer)
	} else {
		// Prompt for OIDC server root with default value
		oidcPrompt := fmt.Sprintf("Enter the OIDC server root [%s]", c.OIDCServer)
		inputOIDC, _ := pterm.DefaultInteractiveTextInput.Show(oidcPrompt)
		
		// Use default if user didn't enter anything
		inputOIDC = strings.TrimSpace(inputOIDC)
		if inputOIDC == "" {
			oidcServer = c.OIDCServer
		} else {
			oidcServer = inputOIDC
		}
		pterm.Info.Printf("Using OIDC server at: %s\n", oidcServer)
	}

	// Perform OIDC authentication
	ctx := context.Background()
	if err := oidc.Login(ctx, baseURL, oidcServer); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	
	// Get authenticated client for API calls
	authClient, err := oidc.GetAuthenticatedClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authenticated client: %w", err)
	}
	
	// Create API client
	apiClient := api.NewClient(baseURL, authClient)
	
	pterm.Info.Println("\nChecking user permissions...")
	
	// List user permissions first
	permissions, err := apiClient.ListPermissions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user permissions: %w", err)
	}
	
	// Display user permissions
	if len(permissions) == 0 {
		pterm.Warning.Println("No permissions found for current user")
	} else {
		pterm.Info.Printf("Found %d permission(s):\n", len(permissions))
		for _, perm := range permissions {
			pterm.Info.Printf("  • %s (%s: %s)\n", perm.PermissionType, perm.Scope, perm.ScopeID)
		}
	}
	
	pterm.Info.Println("\nChecking instance configuration...")
	
	// Get instance configuration to determine edition
	instanceConfig, err := apiClient.GetInstanceConfiguration(ctx)
	if err != nil {
		return fmt.Errorf("failed to get instance configuration: %w", err)
	}
	
	pterm.Info.Printf("Instance edition: %s\n", instanceConfig.Edition)
	
	// For enterprise edition, check instance admin permissions early
	if instanceConfig.Edition == "enterprise" {
		pterm.Info.Println("Enterprise edition detected - checking instance admin permissions...")
		hasInstanceAdmin, err := apiClient.HasInstanceAdminPermission(ctx)
		if err != nil {
			return fmt.Errorf("failed to check instance admin permissions: %w", err)
		}
		if !hasInstanceAdmin {
			return fmt.Errorf("insufficient permissions: enterprise edition requires 'instance_admin' permissions to manage regions and dataplanes")
		}
		pterm.Success.Println("✓ Instance admin permissions confirmed")
	}
	
	// Get user's organizations first
	pterm.Info.Println("Fetching your organizations...")
	organizations, err := apiClient.ListOrganizations(ctx)
	if err != nil {
		return fmt.Errorf("failed to list organizations: %w", err)
	}
	
	// Determine which organization to use
	var selectedOrganization *api.Organization
	
	if len(organizations) == 0 {
		return fmt.Errorf("no organizations found for your account")
	} else if len(organizations) == 1 {
		// If only one organization, use it automatically
		selectedOrganization = organizations[0]
		pterm.Info.Printf("Using organization: %s\n", selectedOrganization.Name)
	} else {
		// If multiple organizations, let user choose
		orgOptions := make([]string, len(organizations))
		orgMap := make(map[string]*api.Organization)
		for i, org := range organizations {
			orgOptions[i] = fmt.Sprintf("%s (%s)", org.Name, org.ID)
			orgMap[orgOptions[i]] = org
		}
		
		orgSelectPrompt := pterm.DefaultInteractiveSelect.
			WithOptions(orgOptions).
			WithDefaultText("Select an organization for this data plane")
		
		selectedOrgOption, _ := orgSelectPrompt.Show()
		selectedOrganization = orgMap[selectedOrgOption]
		pterm.Info.Printf("Selected organization: %s\n", selectedOrganization.Name)
	}
	
	// Validate permissions for the selected organization
	pterm.Info.Printf("Validating permissions for organization %s...\n", selectedOrganization.Name)
	if err := apiClient.ValidateDataPlanePermissions(ctx, selectedOrganization.ID, instanceConfig.Edition); err != nil {
		return fmt.Errorf("permission validation failed: %w", err)
	}
	pterm.Success.Println("✓ Permissions validated - you can create dataplanes for this organization")
	
	// Get all regions accessible to the user for the selected organization
	pterm.Info.Printf("Fetching available regions for organization %s...\n", selectedOrganization.Name)
	allRegions, err := apiClient.ListRegions(ctx, selectedOrganization.ID)
	if err != nil {
		return fmt.Errorf("failed to list regions: %w", err)
	}
	
	// Filter regions based on edition (API call already filtered by organization)
	var availableRegions []*api.Region
	publicRegionOrgID := "00000000-00000000-00000000-00000000"
	
	if instanceConfig.Edition == "cloud" {
		// For cloud edition, filter out public regions (API already filtered by org)
		for _, region := range allRegions {
			if region.OrganizationID != publicRegionOrgID {
				availableRegions = append(availableRegions, region)
			}
		}
		pterm.Info.Printf("Found %d private regions for organization %s\n", len(availableRegions), selectedOrganization.Name)
	} else {
		// For enterprise or oss, use all regions returned by API (already filtered by org)
		availableRegions = allRegions
		pterm.Info.Printf("Found %d regions for organization %s\n", len(availableRegions), selectedOrganization.Name)
	}
	
	// Build region selection options
	regionOptions := []string{"➕ Create New Region"}
	regionMap := make(map[string]*api.Region)
	
	for _, region := range availableRegions {
		optionText := fmt.Sprintf("%s", region.Name)
		regionOptions = append(regionOptions, optionText)
		regionMap[optionText] = region
	}
	
	// Show region selection
	regionSelectPrompt := pterm.DefaultInteractiveSelect.
		WithOptions(regionOptions).
		WithDefaultText("Select a region for the new data plane")
	
	selectedOption, _ := regionSelectPrompt.Show()
	
	var regionID string
	organizationID := selectedOrganization.ID
	
	if selectedOption == "➕ Create New Region" {
		// Create new region using the already selected organization
		regionNamePrompt := "Enter a name for the new region"
		regionName, _ := pterm.DefaultInteractiveTextInput.Show(regionNamePrompt)
		regionName = strings.TrimSpace(regionName)
		
		if regionName == "" {
			return fmt.Errorf("region name is required")
		}
		
		pterm.Info.Printf("Creating region '%s' in organization %s...\n", regionName, selectedOrganization.Name)
		region, err := apiClient.CreateRegion(ctx, &api.CreateRegionRequest{
			Name:           regionName,
			OrganizationID: organizationID,
		})
		if err != nil {
			return fmt.Errorf("failed to create region: %w", err)
		}
		
		regionID = region.ID
		pterm.Success.Printf("Region created successfully! ID: %s\n", regionID)
	} else {
		// Use existing region
		selectedRegion := regionMap[selectedOption]
		regionID = selectedRegion.ID
		pterm.Info.Printf("Using existing region: %s (ID: %s)\n", selectedRegion.Name, regionID)
		pterm.Debug.Printf("Debug - Selected region struct: %+v\n", selectedRegion)
	}
	
	// Create data plane
	dataPlaneNamePrompt := "Enter a name for the data plane"
	dataPlaneName, _ := pterm.DefaultInteractiveTextInput.Show(dataPlaneNamePrompt)
	dataPlaneName = strings.TrimSpace(dataPlaneName)
	
	if dataPlaneName == "" {
		return fmt.Errorf("data plane name is required")
	}
	
	pterm.Info.Printf("Creating data plane '%s'...\n", dataPlaneName)
	
	// Debug: Print the values being sent
	pterm.Debug.Printf("Debug - Data plane name: '%s'\n", dataPlaneName)
	pterm.Debug.Printf("Debug - Region ID: '%s'\n", regionID)
	
	if regionID == "" {
		return fmt.Errorf("region ID is empty - cannot create data plane")
	}
	
	request := &api.CreateDataPlaneRequest{
		Name:     dataPlaneName,
		RegionID: regionID,
	}
	
	pterm.Debug.Printf("Debug - Request: Name='%s', RegionID='%s'\n", request.Name, request.RegionID)
	
	response, err := apiClient.CreateDataPlane(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to create data plane: %w", err)
	}
	
	pterm.Success.Printf("Data plane created successfully!\n")
	pterm.Info.Printf("Data Plane ID: %s\n", response.DataPlaneID)
	pterm.Info.Printf("Client ID: %s\n", response.ClientID)
	pterm.Info.Printf("Client Secret: %s\n", response.ClientSecret)
	pterm.Info.Printf("Region ID: %s\n", regionID)
	
	// Save dataplane credentials to the credentials file
	pterm.Info.Println("\nSaving dataplane credentials...")
	dataplaneInfo := &oidc.DataPlaneInfo{
		DataPlaneID:    response.DataPlaneID,
		ClientID:       response.ClientID,
		ClientSecret:   response.ClientSecret,
		RegionID:       regionID,
		Name:           dataPlaneName,
		OrganizationID: organizationID,
	}
	
	if err := oidc.SaveDataPlaneInfo(dataplaneInfo); err != nil {
		pterm.Warning.Printf("Failed to save dataplane credentials: %v\n", err)
		pterm.Warning.Println("⚠️  Please save the following information manually:")
		pterm.Info.Printf("  Data Plane ID: %s\n", response.DataPlaneID)
		pterm.Info.Printf("  Client ID: %s\n", response.ClientID)
		pterm.Info.Printf("  Client Secret: %s\n", response.ClientSecret)
	} else {
		pterm.Success.Println("✓ Dataplane credentials saved to ~/.abctl/credentials")
	}
	
	// Save configuration for future use
	pterm.Info.Println("\nYour data plane has been created. You can now use it to configure workspaces.")
	
	return nil
}
