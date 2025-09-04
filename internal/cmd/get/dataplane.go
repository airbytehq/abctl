package get

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/api"
	"github.com/airbytehq/abctl/internal/http"
)

// DataplaneCmd handles getting dataplane details.
type DataplaneCmd struct {
	ID     string `arg:"" optional:"" help:"ID of the dataplane (optional, lists all if not specified)."`
	Output string `short:"o" enum:"json,yaml,table" default:"table" help:"Output format (json, yaml, table)."`
}

// Run executes the get dataplane command.
func (c *DataplaneCmd) Run(ctx context.Context, httpClient http.HTTPDoer, apiFactory api.Factory, cfg airbox.ConfigProvider) error {
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

	if c.ID != "" {
		// Get dataplane by ID
		dataplane, err := apiClient.GetDataplane(ctx, c.ID)
		if err != nil {
			return err
		}

		jsonData, err := json.MarshalIndent(dataplane, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal response: %w", err)
		}

		fmt.Println(string(jsonData))
	} else {
		// List all dataplanes
		dataplanes, err := apiClient.ListDataplanes(ctx)
		if err != nil {
			return fmt.Errorf("failed to list dataplanes: %w", err)
		}

		jsonData, err := json.MarshalIndent(dataplanes, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal response: %w", err)
		}

		fmt.Println(string(jsonData))
	}

	return nil
}
