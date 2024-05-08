package local

import (
	"fmt"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/local"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewCmdUninstall() *cobra.Command {
	spinner := &pterm.DefaultSpinner

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Airbyte locally",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			spinner, _ = spinner.Start("Starting uninstallation")
			spinner.UpdateText("Checking for Docker installation")

			dockerVersion, err := dockerInstalled(cmd.Context())
			if err != nil {
				pterm.Error.Println("Unable to determine if Docker is installed")
				return fmt.Errorf("could not determine docker installation status: %w", err)
			}

			telClient.Attr("docker_version", dockerVersion.Version)
			telClient.Attr("docker_arch", dockerVersion.Arch)
			telClient.Attr("docker_platform", dockerVersion.Platform)

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return telemetry.Wrapper(cmd.Context(), telemetry.Uninstall, func() error {
				spinner.UpdateText(fmt.Sprintf("Checking for existing Kubernetes cluster '%s'", provider.ClusterName))

				cluster, err := k8s.NewCluster(provider)
				if err != nil {
					pterm.Error.Printfln("Could not determine if the cluster '%s' exists", provider.ClusterName)
					return err
				}

				// if no cluster exists, there is nothing to do
				if !cluster.Exists() {
					pterm.Success.Printfln("Cluster '%s' does not exist\nNo additional action required", provider.ClusterName)
					return nil
				}

				pterm.Success.Printfln("Existing cluster '%s' found", provider.ClusterName)

				lc, err := local.New(provider, flagPort, local.WithTelemetryClient(telClient), local.WithSpinner(spinner))
				if err != nil {
					pterm.Warning.Printfln("Failed to initialize 'local' command\nUninstallation attempt will continue")
					pterm.Debug.Printfln("Initialization of 'local' failed with %s", err.Error())
				} else {
					if err := lc.Uninstall(cmd.Context()); err != nil {
						pterm.Warning.Printfln("could not complete uninstall: %s", err.Error())
						pterm.Warning.Println("will still attempt to uninstall the cluster")
					}
				}

				spinner.UpdateText(fmt.Sprintf("Verifying uninstallation status of cluster '%s'", provider.ClusterName))
				if err := cluster.Delete(); err != nil {
					pterm.Error.Printfln("Uninstallation of cluster '%s' failed", provider.ClusterName)
					return fmt.Errorf("could not uninstall cluster %s", provider.ClusterName)
				}
				pterm.Success.Printfln("Uninstallation of cluster '%s' completed successfully", provider.ClusterName)

				spinner.Success("Airbyte uninstallation complete")

				return nil
			})
		},
	}

	return cmd
}
