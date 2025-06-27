package local

import (
	"context"
	"fmt"
	"os"

	"github.com/airbytehq/abctl/internal/paths"
	"github.com/pterm/pterm"
)

type UninstallOpts struct {
	Persisted bool
}

// Uninstall handles the uninstallation of Airbyte.
func (c *Command) Uninstall(_ context.Context, opts UninstallOpts) error {
	// check if persisted data should be removed, if not this is a noop
	if opts.Persisted {
		c.spinner.UpdateText("Removing persisted data")
		if err := os.RemoveAll(paths.Data); err != nil {
			pterm.Error.Println(fmt.Sprintf("Unable to remove persisted data '%s'", paths.Data))
			return fmt.Errorf("unable to remove persisted data '%s': %w", paths.Data, err)
		}
		pterm.Success.Println("Removed persisted data")
	}

	return nil
}
