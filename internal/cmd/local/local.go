package local

import (
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

// telClient is the telemetry telClient to use.
// This will be set in the persistentPreRunLocal method which runs prior to any commands being executed.
var telClient telemetry.Client

// provider is which provider is being used.
// This will be set in the persistentPreRunLocal method which runs prior to any commands being executed.
var provider k8s.Provider

// NewCmdLocal represents the local command.
func NewCmdLocal() *cobra.Command {
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
			// provider configuration
			// TODO: cleanup
			{
				provider = k8s.KindProvider
				printK8sProvider(provider)
			}

			return nil
		},
		Short: "Manages local Airbyte installations",
	}
	cmd.AddCommand(NewCmdInstall(), NewCmdUninstall())

	return cmd
}

func printK8sProvider(p k8s.Provider) {
	userHome, _ := os.UserHomeDir()
	configPath := filepath.Join(userHome, p.Kubeconfig)
	pterm.Info.Printfln("Using Kubernetes provider:\n\tProvider: %s\n\tKubeconfig: %s\n\tContext: %s", p.Name, configPath, p.Context)
}
