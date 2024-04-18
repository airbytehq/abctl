package version

import (
	"github.com/airbytehq/abctl/internal/build"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// Cmd returns a cobra command for printing the version information.
// The version information is read directly from build.Version.
var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		pterm.Printfln(`version: %s
revision: %s
time: %s
modified: %t`, build.Version, build.Revision, build.ModificationTime, build.Modified)
	},
}
