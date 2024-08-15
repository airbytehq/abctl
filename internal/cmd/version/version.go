package version

import (
	"fmt"
	"strings"

	"github.com/airbytehq/abctl/internal/build"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// NewCmdVersion returns a cobra command for printing the version information.
// The version information is read directly from build.Version.
func NewCmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			parts := []string{fmt.Sprintf("version: %s", build.Version)}
			if build.Revision != "" {
				parts = append(parts, fmt.Sprintf("revision: %s", build.Revision))
			}
			if build.ModificationTime != "" {
				parts = append(parts, fmt.Sprintf("time: %s", build.ModificationTime))
			}
			if build.Modified {
				parts = append(parts, fmt.Sprintf("modified: %t", build.Modified))
			}
			pterm.Println(strings.Join(parts, "\n"))
		},
	}
}
