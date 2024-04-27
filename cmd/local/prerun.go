package local

import (
	"fmt"
	"github.com/airbytehq/abctl/internal/local/k8s"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

// persistentPreRunLocal is extracted from the cobra.Command initialization for reading purposes only
func persistentPreRunLocal(cmd *cobra.Command, _ []string) error {
	// telemetry client configuration
	{
		// ignore the error as it will default to false if an error returns
		dnt, _ := cmd.Flags().GetBool("dnt")
		var telOpts []telemetry.GetOption
		if dnt {
			telOpts = append(telOpts, telemetry.WithDnt())
		}

		telClient = telemetry.Get(telOpts...)
	}
	// provider configuration
	{
		var err error
		provider, err = k8s.ProviderFromString(flagProvider)
		if err != nil {
			return err
		}

		printK8sProvider(provider)
	}

	return nil
}

func printK8sProvider(p k8s.Provider) {
	userHome, _ := os.UserHomeDir()
	configPath := filepath.Join(userHome, p.Kubeconfig)
	pterm.Info.Printfln("using kubernetes provider:\n  provider name: %s\n  kubeconfig: %s\n  context: %s",
		p.Name, configPath, p.Context)
}

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
