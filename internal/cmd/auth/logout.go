package auth

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/ui"
)

// LogoutCmd handles logout and credential cleanup
type LogoutCmd struct {
	Namespace string `short:"n" help:"Target namespace (default: current kubeconfig context)."`
}

// Run executes the logout command
func (c *LogoutCmd) Run(ctx context.Context, cfgStore airbox.ConfigStore, ui ui.Provider) error {
	ui.Title("Logging out of Airbyte")

	// Load airbox config
	cfg, err := cfgStore.Load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return airbox.NewConfigInitError("no airbox configuration found")
		}
		return fmt.Errorf("failed to load airbox config: %w", err)
	}

	// Check if user credentials exist
	if !cfg.IsAuthenticated() {
		return airbox.NewLoginError("not authenticated")
	}

	// Clear only the tokens, keep user identity intact
	cfg.Credentials = nil

	// Save updated config
	if err := cfgStore.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	ui.ShowSuccess("Successfully logged out!")
	ui.NewLine()

	return nil
}
