package local

import (
	"fmt"
	"github.com/airbytehq/abctl/internal/cmd/local/docker"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/local"
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

	// envDockerServer is the env-var that can be specified to override the default docker registry.
	envDockerServer = "ABCTL_LOCAL_INSTALL_DOCKER_SERVER"
	// envDockerUser is the env-var that can be specified to override the default docker username.
	envDockerUser = "ABCTL_LOCAL_INSTALL_DOCKER_USERNAME"
	// envDockerPass is the env-var that can be specified to override the default docker password.
	envDockerPass = "ABCTL_LOCAL_INSTALL_DOCKER_PASSWORD"
	// envDockerEmail is the env-var that can be specified to override the default docker email.
	envDockerEmail = "ABCTL_LOCAL_INSTALL_DOCKER_EMAIL"
)

func NewCmdInstall(provider k8s.Provider) *cobra.Command {
	spinner := &pterm.DefaultSpinner

	var (
		flagBasicAuthUser string
		flagBasicAuthPass string

		flagChartValuesFile string
		flagChartVersion    string
		flagMigrate         bool
		flagPort            int

		flagDockerServer string
		flagDockerUser   string
		flagDockerPass   string
		flagDockerEmail  string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Airbyte locally",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			spinner, _ = spinner.Start("Starting installation")
			spinner.UpdateText("Checking for Docker installation")

			dockerVersion, err := dockerInstalled(cmd.Context())
			if err != nil {
				pterm.Error.Println("Unable to determine if Docker is installed")
				return fmt.Errorf("unable to determine docker installation status: %w", err)
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
			return telClient.Wrap(cmd.Context(), telemetry.Install, func() error {
				spinner.UpdateText(fmt.Sprintf("Checking for existing Kubernetes cluster '%s'", provider.ClusterName))

				cluster, err := provider.Cluster()
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
							dockerClient, err = docker.New(cmd.Context())
							if err != nil {
								pterm.Error.Printfln("Could not connect to Docker daemon")
								return fmt.Errorf("unable to connect to docker: %w", err)
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

				lc, err := local.New(provider,
					local.WithPortHTTP(flagPort),
					local.WithTelemetryClient(telClient),
					local.WithSpinner(spinner),
				)
				if err != nil {
					pterm.Error.Printfln("Failed to initialize 'local' command")
					return fmt.Errorf("unable to initialize local command: %w", err)
				}

				opts := local.InstallOpts{
					BasicAuthUser:    flagBasicAuthUser,
					BasicAuthPass:    flagBasicAuthPass,
					HelmChartVersion: flagChartVersion,
					ValuesFile:       flagChartValuesFile,
					Migrate:          flagMigrate,
					Docker:           dockerClient,

					DockerServer: flagDockerServer,
					DockerUser:   flagDockerUser,
					DockerPass:   flagDockerPass,
					DockerEmail:  flagDockerEmail,
				}

				if opts.HelmChartVersion == "latest" {
					opts.HelmChartVersion = ""
				}

				if env := os.Getenv(envBasicAuthUser); env != "" {
					opts.BasicAuthUser = env
				}
				if env := os.Getenv(envBasicAuthPass); env != "" {
					opts.BasicAuthPass = env
				}

				if env := os.Getenv(envDockerServer); env != "" {
					opts.DockerServer = env
				}
				if env := os.Getenv(envDockerUser); env != "" {
					opts.DockerUser = env
				}
				if env := os.Getenv(envDockerPass); env != "" {
					opts.DockerPass = env
				}
				if env := os.Getenv(envDockerEmail); env != "" {
					opts.DockerEmail = env
				}

				if err := lc.Install(cmd.Context(), opts); err != nil {
					spinner.Fail("Unable to install Airbyte locally")
					return err
				}

				spinner.Success("Airbyte installation complete")
				return nil
			})
		},
	}

	cmd.FParseErrWhitelist.UnknownFlags = true

	cmd.Flags().StringVarP(&flagBasicAuthUser, "username", "u", "airbyte", "basic auth username, can also be specified via "+envBasicAuthUser)
	cmd.Flags().StringVarP(&flagBasicAuthPass, "password", "p", "password", "basic auth password, can also be specified via "+envBasicAuthPass)
	cmd.Flags().IntVar(&flagPort, "port", local.Port, "ingress http port")

	cmd.Flags().StringVar(&flagChartVersion, "chart-version", "latest", "specify the Airbyte helm chart version to install")
	cmd.Flags().StringVar(&flagChartValuesFile, "values", "", "the Airbyte helm chart values file to load")
	cmd.Flags().BoolVar(&flagMigrate, "migrate", false, "migrate data from docker compose installation")

	cmd.Flags().StringVar(&flagDockerServer, "docker-server", "https://index.docker.io/v1/", "docker registry, can also be specified via "+envDockerServer)
	cmd.Flags().StringVar(&flagDockerUser, "docker-username", "", "docker username, can also be specified via "+envDockerEmail)
	cmd.Flags().StringVar(&flagDockerPass, "docker-password", "", "docker password, can also be specified via "+envDockerPass)
	cmd.Flags().StringVar(&flagDockerEmail, "docker-email", "", "docker email, can also be specified via "+envDockerEmail)

	cmd.MarkFlagsRequiredTogether("docker-username", "docker-password", "docker-email")

	return cmd
}
