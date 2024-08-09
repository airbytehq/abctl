package local

import (
	"errors"
	"fmt"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/cmd/local/paths"
	"github.com/airbytehq/abctl/internal/status"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"io/fs"
	"os"
)

var telClient telemetry.Client

var blue = lipgloss.NewStyle().Foreground(lipgloss.Color("94"))

// NewCmdLocal represents the local command.
func NewCmdLocal(provider k8s.Provider) *cobra.Command {
	cmd := &cobra.Command{
		Use: "local",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := checkAirbyteDir(); err != nil {
				return fmt.Errorf("%w: %w", localerr.ErrAirbyteDir, err)
			}

			telClient = telemetry.Get()

			{
				// show the deprecation warning for username and password
				userFlag, _ := cmd.Flags().GetString("username")
				passFlag, _ := cmd.Flags().GetString("password")
				if (userFlag != "" && userFlag != "airbyte") || (passFlag != "" && passFlag != "password") {
					status.Warn("The --username and --password flags have been deprecated.\n" +
						"  Credentials now are randomly generated and can be retrieved by running\n" +
						blue.Render("  abctl local credentials"))
				}
			}
			printProviderDetails(provider)

			return nil
		},
		Short: "Manages local Airbyte installations",
	}

	cmd.AddCommand(NewCmdInstall(provider), NewCmdUninstall(provider), NewCmdStatus(provider), NewCmdCredentials(provider))

	return cmd
}

func printProviderDetails(p k8s.Provider) {
	status.Info(fmt.Sprintf(
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
