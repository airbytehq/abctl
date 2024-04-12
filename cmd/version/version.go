package version

import (
	"airbyte.io/abctl/internal/build"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		pterm.Printfln("version: %s", build.Version)
	},
}
