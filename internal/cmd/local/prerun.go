package local

import (
	"fmt"
	"github.com/spf13/cobra"
)

// preRunInstall is extracted from the cobra.Command initialization for reading purposes only
func preRunInstall(cmd *cobra.Command, _ []string) error {
	if err := dockerInstalled(cmd.Context(), telClient); err != nil {
		return fmt.Errorf("could not determine docker installation status: %w", err)
	}

	if err := portAvailable(cmd.Context(), flagPort); err != nil {
		return fmt.Errorf("port %d is not available: %w", flagPort, err)
	}

	return nil
}

// preRunUninstall is extracted from the cobra.Command initialization for reading purposes only
func preRunUninstall(cmd *cobra.Command, _ []string) error {
	if err := dockerInstalled(cmd.Context(), telClient); err != nil {
		return fmt.Errorf("could not determine docker installation status: %w", err)
	}

	return nil
}
