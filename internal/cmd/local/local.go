package local

import (
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/paths"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"path/filepath"
)

var telClient telemetry.Client

// NewCmdLocal represents the local command.
func NewCmdLocal(provider k8s.Provider) *cobra.Command {
	cmd := &cobra.Command{
		Use: "local",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// telemetry client configuration
			{
				var telOpts []telemetry.GetOption
				// This is deprecated as the telemetry.Client now checks itself if the DO_NOT_TRACK env-var is defined.
				// Currently leaving this here to output the message about the --dnt trag no longer being supported.
				dntFlag, _ := cmd.Flags().GetBool("dnt")
				if dntFlag {
					pterm.Warning.Println("The --dnt flag has been deprecated. Use DO_NOT_TRACK environment-variable instead.")
				}

				telClient = telemetry.Get(telOpts...)
			}
			printProviderDetails(provider)

			return nil
		},
		Short: "Manages local Airbyte installations",
	}

	cmd.AddCommand(NewCmdInstall(provider), NewCmdUninstall(provider), NewCmdStatus(provider))

	return cmd
}

func printProviderDetails(p k8s.Provider) {
	configPath := filepath.Join(paths.UserHome, p.Kubeconfig)
	pterm.Info.Printfln("Using Kubernetes provider:\n  Provider: %s\n  Kubeconfig: %s\n  Context: %s", p.Name, configPath, p.Context)
}
