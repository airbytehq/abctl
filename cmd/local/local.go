package local

import (
	"airbyte.io/abctl/internal/local"
	"fmt"

	"github.com/spf13/cobra"
)

// Cmd represents the local command
var Cmd = &cobra.Command{
	Use:   "local",
	Short: "Manages local Airbyte installations",
}

var InstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Airbyte locally",
	RunE: func(cmd *cobra.Command, args []string) error {
		lc, err := local.New()
		if err != nil {
			return fmt.Errorf("could not initialize local client: %w", err)
		}

		return lc.Install()
	},
}

var UninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Airbyte locally",
	RunE: func(cmd *cobra.Command, args []string) error {
		lc, err := local.New()
		if err != nil {
			return fmt.Errorf("could not initialize local client: %w", err)
		}

		return lc.Uninstall()
	},
}

func init() {
	Cmd.AddCommand(InstallCmd)
	Cmd.AddCommand(UninstallCmd)
}
