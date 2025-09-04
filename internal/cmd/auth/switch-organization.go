package auth

import (
	"context"
	"fmt"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/api"
	"github.com/airbytehq/abctl/internal/http"
	"github.com/airbytehq/abctl/internal/ui"
)

// SwitchOrganizationCmd handles switching between orgs.
type SwitchOrganizationCmd struct{}

// Run executes the switch organization command.
func (c *SwitchOrganizationCmd) Run(ctx context.Context, httpClient http.HTTPDoer, cfg airbox.ConfigProvider, ui ui.Provider) error {
	ui.Title("Switching Workspace")
	ui.ShowInfo("Switch between your user's organizations.")
	ui.NewLine()

	// Load airbox config and current context
	abCfg, err := cfg.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if user is authenticated
	if err := abCfg.IsAuthenticated(); err != nil {
		return err
	}

	// Get current context
	currentContext, err := abCfg.GetCurrentContext()
	if err != nil {
		return fmt.Errorf("could not get current context from airbox configuration: %w", err)
	}

	// Create API client
	apiClient, err := api.NewFactory(ctx, cfg, httpClient)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Get available orgs
	var org *api.Organization

	orgs, err := apiClient.ListOrganizations(ctx)
	if err != nil {
		return fmt.Errorf("failed to list orgs: %w", err)
	}

	switch len(orgs) {
	case 0:
		return fmt.Errorf("no organizations found")
	case 1:
		return fmt.Errorf("you belong to a single organization. switching not possible")
	}

	orgNames := make([]string, len(orgs))
	for i, org := range orgs {
		orgNames[i] = org.Name
	}

	// Use filterable select for better UX when there are many organizations
	var idx int
	if len(orgs) > 10 {
		idx, _, err = ui.FilterableSelect("Select an organization", orgNames)
	} else {
		idx, _, err = ui.Select("Select an organization:", orgNames)
	}
	if err != nil {
		return err
	}

	org = orgs[idx]

	currentContext.OrganizationID = org.ID

	abCfg.AddContext(abCfg.CurrentContext, *currentContext)
	if err := cfg.Save(abCfg); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	err = cfg.Save(abCfg)
	if err != nil {
		return err
	}

	ui.ShowSuccess("Organization switched successfully!")
	ui.NewLine()

	return nil
}
