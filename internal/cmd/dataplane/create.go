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
}

func (c *CreateCmd) Run() error {
	// Prompt for base URL with default value
	defaultURL := "https://cloud.airbyte.com"

	urlPrompt := fmt.Sprintf("Enter the base URL of the Airbyte instance [%s]", defaultURL)
	baseURL, _ := pterm.DefaultInteractiveTextInput.Show(urlPrompt)

	// Use default if user didn't enter anything
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultURL
	}

	pterm.Info.Printf("Using Airbyte instance at: %s\n", baseURL)

	// Prompt for OIDC server root with default value
	defaultOIDCServer := "https://cloud.airbyte.com/auth"
	
	oidcPrompt := fmt.Sprintf("Enter the OIDC server root [%s]", defaultOIDCServer)
	oidcServer, _ := pterm.DefaultInteractiveTextInput.Show(oidcPrompt)
	
	// Use default if user didn't enter anything
	oidcServer = strings.TrimSpace(oidcServer)
	if oidcServer == "" {
		oidcServer = defaultOIDCServer
	}
	
	pterm.Info.Printf("Using OIDC server at: %s\n", oidcServer)

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
	
	// Prompt for organization ID
	pterm.Info.Println("\nNow let's create a data plane configuration.")
	organizationIDPrompt := "Enter your Organization ID (UUID)"
	organizationID, _ := pterm.DefaultInteractiveTextInput.Show(organizationIDPrompt)
	organizationID = strings.TrimSpace(organizationID)
	
	if organizationID == "" {
		return fmt.Errorf("organization ID is required")
	}
	
	// Ask if user wants to create a new region or use existing
	useExistingRegion := false
	regionSelectPrompt := pterm.DefaultInteractiveSelect.
		WithOptions([]string{"Create new region", "Use existing region"}).
		WithDefaultText("Would you like to create a new region or use an existing one?")
	
	regionChoice, _ := regionSelectPrompt.Show()
	useExistingRegion = (regionChoice == "Use existing region")
	
	var regionID string
	
	if useExistingRegion {
		// List existing regions
		pterm.Info.Println("Fetching existing regions...")
		regions, err := apiClient.ListRegions(ctx, organizationID)
		if err != nil {
			return fmt.Errorf("failed to list regions: %w", err)
		}
		
		if len(regions) == 0 {
			pterm.Warning.Println("No existing regions found. Creating a new region...")
			useExistingRegion = false
		} else {
			// Show region selection
			regionOptions := make([]string, len(regions))
			regionMap := make(map[string]string)
			for i, region := range regions {
				regionOptions[i] = fmt.Sprintf("%s (ID: %s)", region.Name, region.ID)
				regionMap[regionOptions[i]] = region.ID
			}
			
			regionSelectPrompt := pterm.DefaultInteractiveSelect.
				WithOptions(regionOptions).
				WithDefaultText("Select a region")
			
			selectedRegion, _ := regionSelectPrompt.Show()
			regionID = regionMap[selectedRegion]
		}
	}
	
	if !useExistingRegion {
		// Create new region
		regionNamePrompt := "Enter a name for the new region"
		regionName, _ := pterm.DefaultInteractiveTextInput.Show(regionNamePrompt)
		regionName = strings.TrimSpace(regionName)
		
		if regionName == "" {
			return fmt.Errorf("region name is required")
		}
		
		pterm.Info.Printf("Creating region '%s'...\n", regionName)
		region, err := apiClient.CreateRegion(ctx, &api.CreateRegionRequest{
			Name:           regionName,
			OrganizationID: organizationID,
		})
		if err != nil {
			return fmt.Errorf("failed to create region: %w", err)
		}
		
		regionID = region.ID
		pterm.Success.Printf("Region created successfully! ID: %s\n", regionID)
	}
	
	// Create data plane
	dataPlaneNamePrompt := "Enter a name for the data plane"
	dataPlaneName, _ := pterm.DefaultInteractiveTextInput.Show(dataPlaneNamePrompt)
	dataPlaneName = strings.TrimSpace(dataPlaneName)
	
	if dataPlaneName == "" {
		return fmt.Errorf("data plane name is required")
	}
	
	pterm.Info.Printf("Creating data plane '%s'...\n", dataPlaneName)
	dataPlane, err := apiClient.CreateDataPlane(ctx, &api.CreateDataPlaneRequest{
		Name:     dataPlaneName,
		RegionID: regionID,
	})
	if err != nil {
		return fmt.Errorf("failed to create data plane: %w", err)
	}
	
	pterm.Success.Printf("Data plane created successfully!\n")
	pterm.Info.Printf("Data Plane ID: %s\n", dataPlane.ID)
	pterm.Info.Printf("Region ID: %s\n", regionID)
	
	// Save configuration for future use
	pterm.Info.Println("\nYour data plane has been created. You can now use it to configure workspaces.")
	
	return nil
}
