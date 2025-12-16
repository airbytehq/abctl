package auth

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/api"
	"github.com/airbytehq/abctl/internal/http"
	"github.com/airbytehq/abctl/internal/ui"
)

// SwitchOrganizationCmd handles switching between orgs.
type SwitchOrganizationCmd struct{}

// Run executes the switch organization command.
func (c *SwitchOrganizationCmd) Run(ctx context.Context, cfgStore airbox.ConfigStore, httpClient http.HTTPDoer, apiFactory airbox.APIServiceFactory, ui ui.Provider) error {
	ui.Title("Switching Workspace")
	ui.ShowInfo("Switch between your user's organizations.")
	ui.NewLine()

	apiClient, err := apiFactory(ctx, httpClient, cfgStore)
	if err != nil {
		return err
	}

	// Load config for saving later
	cfg, err := cfgStore.Load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return airbox.NewConfigInitError("no airbox configuration found")
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current context for saving later
	currentContext, err := cfg.GetCurrentContext()
	if err != nil {
		return fmt.Errorf("could not get current context: %w", err)
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
		// Automatically select the single organization
		org = orgs[0]
		ui.ShowInfo(fmt.Sprintf("Setting organization to: %s", org.Name))
		ui.NewLine()
	default:
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
	}

	currentContext.OrganizationID = org.ID

	cfg.AddContext(cfg.CurrentContext, *currentContext)
	if err := cfgStore.Save(cfg); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	err = cfgStore.Save(cfg)
	if err != nil {
		return err
	}

	ui.ShowSuccess("Organization switched successfully!")
	ui.NewLine()

	return nil
}
