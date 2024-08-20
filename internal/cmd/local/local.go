package local

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/cmd/local/paths"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
)

var telClient telemetry.Client

type Cmd struct {
	Credentials CredentialsCmd `cmd:"" help:"Get local Airbyte user credentials."`
	Install     InstallCmd     `cmd:"" help:"Install local Airbyte."`
	Status      StatusCmd      `cmd:"" help:"Get local Airbyte status."`
	Uninstall   UninstallCmd   `cmd:"" help:"Uninstall local Airbyte."`
}

func (c *Cmd) BeforeApply() error {
	if err := checkAirbyteDir(); err != nil {
		return fmt.Errorf("%w: %w", localerr.ErrAirbyteDir, err)
	}

	telClient = telemetry.Get()
	return nil
}

func (c *Cmd) AfterApply(provider k8s.Provider) error {
	printProviderDetails(provider)
	return nil
}

func printProviderDetails(p k8s.Provider) {
	pterm.Info.Println(fmt.Sprintf(
		"Using Kubernetes provider:\n  Provider: %s\n  Kubeconfig: %s\n  Context: %s",
		p.Name, p.Kubeconfig, p.Context,
	))
}

// checkAirbyteDir verifies that, if the paths.Airbyte directory exists, that it has proper permissions.
// If the directory does not have the proper permissions, this method will attempt to fix them.
// A nil response either indicates that either:
// - no paths.Airbyte directory exists
// - the permissions are already correct
// - this function was able to fix the incorrect permissions.
func checkAirbyteDir() error {
	fileInfo, err := os.Stat(paths.Airbyte)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// nothing to do, directory will be created later on
			return nil
		}
		return fmt.Errorf("unable to determine status of '%s': %w", paths.Airbyte, err)
	}

	if !fileInfo.IsDir() {
		return errors.New(paths.Airbyte + " is not a directory")
	}

	if fileInfo.Mode().Perm() >= 0744 {
		// directory has minimal permissions
		return nil
	}

	if err := os.Chmod(paths.Airbyte, 0744); err != nil {
		return fmt.Errorf("unable to change permissions of '%s': %w", paths.Airbyte, err)
	}

	return nil
}
