package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/ui"
)

// LogoutCmd handles logout and credential cleanup
type LogoutCmd struct {
	Namespace string `short:"n" help:"Target namespace (default: current kubeconfig context)."`
}

// Run executes the logout command
func (c *LogoutCmd) Run(ctx context.Context, cfg airbox.ConfigProvider, ui ui.Provider) error {
	ui.Title("Logging out of Airbyte")

	// Load airbox config
	abCfg, err := cfg.Load()
	if err != nil {
		return fmt.Errorf("failed to load airbox config: %w", err)
	}

	// Check if user credentials exist
	if abCfg.Credentials == nil {
		return fmt.Errorf("not logged in - no credentials found")
	}

	// Clear only the tokens, keep user identity intact
	abCfg.Credentials.AccessToken = ""
	abCfg.Credentials.RefreshToken = ""
	abCfg.Credentials.ExpiresAt = time.Time{}

	// Save updated config
	if err := cfg.Save(abCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	ui.ShowSuccess("Successfully logged out!")
	ui.NewLine()
	return nil
}
