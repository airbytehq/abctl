package local

import (
	"fmt"

	"github.com/airbytehq/abctl/internal/cmd/local/docker"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/local"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewCmdStatus(provider k8s.Provider) *cobra.Command {
	spinner := &pterm.DefaultSpinner

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Status of local Airbyte",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			spinner, _ = spinner.Start("Starting status check")
			spinner.UpdateText("Checking for Docker installation")

			dockerVersion, err := dockerInstalled(cmd.Context())
			if err != nil {
				pterm.Error.Println("Unable to determine if Docker is installed")
				return fmt.Errorf("unable to determine docker installation status: %w", err)
			}

			telClient.Attr("docker_version", dockerVersion.Version)
			telClient.Attr("docker_arch", dockerVersion.Arch)
			telClient.Attr("docker_platform", dockerVersion.Platform)

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return telClient.Wrap(cmd.Context(), telemetry.Status, func() error {
				spinner.UpdateText(fmt.Sprintf("Checking for existing Kubernetes cluster '%s'", provider.ClusterName))

				cluster, err := provider.Cluster()
				if err != nil {
					pterm.Error.Printfln("Unable to determine status of any existing '%s' cluster", provider.ClusterName)
					return err
				}

				if !cluster.Exists() {
					pterm.Warning.Println("Airbyte does not appear to be installed locally")
					return nil
				}

				var port int
				pterm.Success.Printfln("Existing cluster '%s' found", provider.ClusterName)
				spinner.UpdateText(fmt.Sprintf("Validating existing cluster '%s'", provider.ClusterName))

				// only for kind do we need to check the existing port
				if provider.Name == k8s.Kind {
					if dockerClient == nil {
						dockerClient, err = docker.New(cmd.Context())
						if err != nil {
							pterm.Error.Printfln("Unable to connect to Docker daemon")
							return fmt.Errorf("unable to connect to docker: %w", err)
						}
					}

					port, err = dockerClient.Port(cmd.Context(), fmt.Sprintf("%s-control-plane", provider.ClusterName))
					if err != nil {
						pterm.Warning.Printfln("Unable to determine docker port for cluster '%s'", provider.ClusterName)
						return nil
					}
				}

				lc, err := local.New(provider,
					local.WithPortHTTP(port),
					local.WithTelemetryClient(telClient),
					local.WithSpinner(spinner),
				)
				if err != nil {
					pterm.Error.Printfln("Failed to initialize 'local' command")
					return fmt.Errorf("unable to initialize local command: %w", err)
				}

				if err := lc.Status(cmd.Context()); err != nil {
					spinner.Fail("Unable to install Airbyte locally")
					return err
				}

				spinner.Success("Status check")
				return nil
			})
		},
	}

	cmd.FParseErrWhitelist.UnknownFlags = true

	return cmd
}
