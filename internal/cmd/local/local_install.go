package local

import (
	"fmt"
	"github.com/airbytehq/abctl/internal/local"
	"github.com/airbytehq/abctl/internal/local/docker"
	"github.com/airbytehq/abctl/internal/local/k8s"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
)

const (
	// envBasicAuthUser is the env-var that can be specified to override the default basic-auth username.
	envBasicAuthUser = "ABCTL_LOCAL_INSTALL_USERNAME"
	// envBasicAuthPass is the env-var that can be specified to override the default basic-auth password.
	envBasicAuthPass = "ABCTL_LOCAL_INSTALL_PASSWORD"
)

func NewCmdInstall() *cobra.Command {
	spinner := &pterm.DefaultSpinner

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Airbyte locally",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			spinner, _ = spinner.Start("Starting installation")
			spinner.UpdateText("Checking for Docker installation")

			dockerVersion, err := dockerInstalled(cmd.Context())
			if err != nil {
				pterm.Error.Println("Unable to determine if Docker is installed")
				return fmt.Errorf("could not determine docker installation status: %w", err)
			}

			telClient.Attr("docker_version", dockerVersion.Version)
			telClient.Attr("docker_arch", dockerVersion.Arch)
			telClient.Attr("docker_platform", dockerVersion.Platform)

			spinner.UpdateText(fmt.Sprintf("Checking if port %d is available", flagPort))
			if err := portAvailable(cmd.Context(), flagPort); err != nil {
				return fmt.Errorf("port %d is not available: %w", flagPort, err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return telemetry.Wrapper(cmd.Context(), telemetry.Install, func() error {
				spinner.UpdateText(fmt.Sprintf("Checking for existing Kubernetes cluster '%s'", provider.ClusterName))

				cluster, err := k8s.NewCluster(provider)
				if err != nil {
					pterm.Error.Printfln("Could not determine status of any existing '%s' cluster", provider.ClusterName)
					return err
				}

				if cluster.Exists() {
					// existing cluster, validate it
					pterm.Success.Printfln("Existing cluster '%s' found", provider.ClusterName)
					spinner.UpdateText(fmt.Sprintf("Validating existing cluster '%s'", provider.ClusterName))

					// only for kind do we need to check the existing port
					if provider.Name == k8s.Kind {
						if dockerClient == nil {
							dockerClient, err = docker.New()
							if err != nil {
								pterm.Error.Printfln("Could not connect to Docker daemon")
								return fmt.Errorf("could not connect to docker: %w", err)
							}
						}

						providedPort := flagPort
						flagPort, err = dockerClient.Port(cmd.Context(), fmt.Sprintf("%s-control-plane", provider.ClusterName))
						if err != nil {
							pterm.Warning.Printfln("Unable to determine which port the existing cluster was configured to use.\n" +
								"Installation will continue but may ultimately fail, in which case it will be necessarily to uninstall first.")
							// since we can't verify the port is correct, push forward with the provided port
							flagPort = providedPort
						}
						if providedPort != flagPort {
							pterm.Warning.Printfln("The existing cluster was found to be using port %d, which differs from the provided port %d.\n"+
								"The existing port will be used, as changing ports currently requires the existing installation to be uninstalled first.", flagPort, providedPort)
						}
					}

					pterm.Success.Printfln("Cluster '%s' validation complete", provider.ClusterName)
				} else {
					// no existing cluster, need to create one
					pterm.Info.Println(fmt.Sprintf("No existing cluster found, cluster '%s' will be created", provider.ClusterName))
					spinner.UpdateText(fmt.Sprintf("Creating cluster '%s'", provider.ClusterName))
					if err := cluster.Create(flagPort); err != nil {
						pterm.Error.Printfln("Cluster '%s' could not be created", provider.ClusterName)
						return err
					}
					pterm.Success.Printfln("Cluster '%s' created", provider.ClusterName)
				}

				lc, err := local.New(provider, flagPort, local.WithTelemetryClient(telClient), local.WithSpinner(spinner))
				if err != nil {
					pterm.Error.Printfln("Failed to initialize 'local' command")
					return fmt.Errorf("could not initialize local command: %w", err)
				}

				user := flagUsername
				if env := os.Getenv(envBasicAuthUser); env != "" {
					user = env
				}
				pass := flagPassword
				if env := os.Getenv(envBasicAuthPass); env != "" {
					pass = env
				}

				if err := lc.Install(cmd.Context(), user, pass); err != nil {
					spinner.Fail("Unable to install Airbyte locally")
					return err
				}

				spinner.Success("Airbyte installation complete")
				return nil
			})
		},
	}

	cmd.Flags().StringVarP(&flagUsername, "username", "u", "airbyte", "basic auth username, can also be specified via "+envBasicAuthUser)
	cmd.Flags().StringVarP(&flagPassword, "password", "p", "password", "basic auth password, can also be specified via "+envBasicAuthPass)

	return cmd
}
