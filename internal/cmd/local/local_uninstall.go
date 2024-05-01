package local

import (
	"github.com/spf13/cobra"
)

func NewCmdUninstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "uninstall",
		Short:   "Uninstall Airbyte locally",
		PreRunE: preRunUninstall,
		RunE:    runUninstall,
	}

	return cmd
}
