package local

import (
	"fmt"

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

				pterm.Success.Printfln("Existing cluster '%s' found", provider.ClusterName)
				spinner.UpdateText(fmt.Sprintf("Validating existing cluster '%s'", provider.ClusterName))

				port, err := getPort(cmd.Context(), provider)
				if err != nil {
					return err
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
