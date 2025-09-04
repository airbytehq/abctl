package delete

import (
	"context"
	"fmt"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/api"
	"github.com/airbytehq/abctl/internal/http"
	"github.com/airbytehq/abctl/internal/ui"
)

// DataplaneCmd handles dataplane deletion.
type DataplaneCmd struct {
	ID    string `arg:"" required:"" help:"ID of the dataplane to delete."`
	Force bool   `short:"f" help:"Force deletion without confirmation."`
}

// Run executes the delete dataplane command.
func (c *DataplaneCmd) Run(ctx context.Context, httpClient http.HTTPDoer, apiFactory api.Factory, cfg airbox.ConfigProvider, ui ui.Provider) error {
	ui.Title("Deleting dataplane")

	// Check if user is authenticated first
	abCfg, err := cfg.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := abCfg.IsAuthenticated(); err != nil {
		return err
	}

	apiClient, err := apiFactory(ctx, cfg, httpClient)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Delete via API using the ID directly
	err = apiClient.DeleteDataplane(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("failed to delete dataplane: %w", err)
	}

	// Show success
	ui.ShowSuccess(fmt.Sprintf("Dataplane ID '%s' deleted successfully", c.ID))
	ui.NewLine()

	return nil
}
