package local

import (
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

// NewCmdLocal represents the local command.
func NewCmdLocal(provider k8s.Provider) *cobra.Command {
	var telClient telemetry.Client

	cmd := &cobra.Command{
		Use: "local",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
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
			printProviderDetails(provider)

			return nil
		},
		Short: "Manages local Airbyte installations",
	}

	cmd.AddCommand(NewCmdInstall(provider, telClient), NewCmdUninstall(provider, telClient))

	return cmd
}

func printProviderDetails(p k8s.Provider) {
	userHome, _ := os.UserHomeDir()
	configPath := filepath.Join(userHome, p.Kubeconfig)
	pterm.Info.Printfln("Using Kubernetes provider:\n\tProvider: %s\n\tKubeconfig: %s\n\tContext: %s", p.Name, configPath, p.Context)
}
