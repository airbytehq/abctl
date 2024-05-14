package local

import (
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

type Config struct {
	Provider  k8s.Provider
	TelClient telemetry.Client
}

// NewCmdLocal represents the local command.
func NewCmdLocal(cfg *Config) *cobra.Command {
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
				cfg.TelClient = telemetry.Get(telOpts...)
			}
			printProviderDetails(cfg.Provider)

			return nil
		},
		Short: "Manages local Airbyte installations",
	}

	cmd.AddCommand(newCmdInstall(cfg), newCmdUninstall(cfg))

	return cmd
}

func printProviderDetails(p k8s.Provider) {
	userHome, _ := os.UserHomeDir()
	configPath := filepath.Join(userHome, p.Kubeconfig)
	pterm.Info.Printfln("Using Kubernetes provider:\n\tProvider: %s\n\tKubeconfig: %s\n\tContext: %s", p.Name, configPath, p.Context)
}
