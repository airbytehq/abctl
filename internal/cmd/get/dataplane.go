package get

import (
	"context"
	"fmt"
	"sort"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/cmd"
	"github.com/airbytehq/abctl/internal/http"
	"github.com/airbytehq/abctl/internal/ui"
)

// DataplaneCmd handles getting dataplane details.
type DataplaneCmd struct {
	ID     string `arg:"" optional:"" help:"ID of the dataplane (optional, lists all if not specified)."`
	Output string `short:"o" enum:"json,yaml" default:"json" help:"Output format (json, yaml)."`
}

// Run executes the get dataplane command.
func (c *DataplaneCmd) Run(ctx context.Context, cfg airbox.ConfigStore, httpClient http.HTTPDoer, apiFactory airbox.APIServiceFactory, uiProvider ui.Provider) error {
	apiClient, err := apiFactory(ctx, httpClient, cfg)
	if err != nil {
		return err
	}

	if c.ID != "" {
		// Get dataplane by ID
		dataplane, err := apiClient.GetDataplane(ctx, c.ID)
		if err != nil {
			return err
		}
		return cmd.RenderOutput(uiProvider, dataplane, c.Output)
	}

	// List all dataplanes
	dataplanes, err := apiClient.ListDataplanes(ctx)
	if err != nil {
		return fmt.Errorf("failed to list dataplanes: %w", err)
	}

	// Sort dataplanes by name for consistent output
	sort.Slice(dataplanes, func(i, j int) bool {
		return dataplanes[i].Name < dataplanes[j].Name
	})

	return cmd.RenderOutput(uiProvider, dataplanes, c.Output)
}
