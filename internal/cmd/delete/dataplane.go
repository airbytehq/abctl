package delete

import (
	"context"
	"fmt"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/http"
	"github.com/airbytehq/abctl/internal/ui"
)

// DataplaneCmd handles dataplane deletion.
type DataplaneCmd struct {
	ID    string `arg:"" required:"" help:"ID of the dataplane to delete."`
	Force bool   `short:"f" help:"Force deletion without confirmation."`
}

// Run executes the delete dataplane command.
func (c *DataplaneCmd) Run(ctx context.Context, cfg airbox.ConfigStore, httpClient http.HTTPDoer, apiFactory airbox.APIServiceFactory, ui ui.Provider) error {
	ui.Title("Deleting dataplane")

	apiClient, err := apiFactory(ctx, httpClient, cfg)
	if err != nil {
		return err
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
