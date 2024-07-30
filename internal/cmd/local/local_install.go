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
		flagChartSecrets    []string
		flagChartVersion    string
		flagMigrate         bool
		flagPort            int
		flagHost            string

		flagDockerServer string
		flagDockerUser   string
		flagDockerPass   string
		flagDockerEmail  string

		flagNoBrowser bool
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
					pterm.Error.Printfln("Unable to determine status of any existing '%s' cluster", provider.ClusterName)
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
								pterm.Error.Printfln("Unable to connect to Docker daemon")
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
					Secrets:          flagChartSecrets,
					Migrate:          flagMigrate,
					Docker:           dockerClient,
					Host:             flagHost,

					DockerServer: flagDockerServer,
					DockerUser:   flagDockerUser,
					DockerPass:   flagDockerPass,
					DockerEmail:  flagDockerEmail,

					NoBrowser: flagNoBrowser,
				}

				if opts.HelmChartVersion == "latest" {
					opts.HelmChartVersion = ""
				}

				envOverride(&opts.BasicAuthUser, envBasicAuthUser)
				envOverride(&opts.BasicAuthPass, envBasicAuthPass)
				envOverride(&opts.DockerServer, envDockerServer)
				envOverride(&opts.DockerUser, envDockerUser)
				envOverride(&opts.DockerPass, envDockerPass)
				envOverride(&opts.DockerEmail, envDockerEmail)

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
	cmd.Flags().StringVar(&flagHost, "host", "localhost", "ingress http host")

	cmd.Flags().StringVar(&flagChartVersion, "chart-version", "latest", "specify the Airbyte helm chart version to install")
	cmd.Flags().StringVar(&flagChartValuesFile, "values", "", "the Airbyte helm chart values file to load")
	cmd.Flags().StringSliceVar(&flagChartSecrets, "secret", []string{}, "an Airbyte helm chart secret file")
	cmd.Flags().BoolVar(&flagMigrate, "migrate", false, "migrate data from docker compose installation")

	cmd.Flags().StringVar(&flagDockerServer, "docker-server", "https://index.docker.io/v1/", "docker registry, can also be specified via "+envDockerServer)
	cmd.Flags().StringVar(&flagDockerUser, "docker-username", "", "docker username, can also be specified via "+envDockerEmail)
	cmd.Flags().StringVar(&flagDockerPass, "docker-password", "", "docker password, can also be specified via "+envDockerPass)
	cmd.Flags().StringVar(&flagDockerEmail, "docker-email", "", "docker email, can also be specified via "+envDockerEmail)

	cmd.Flags().BoolVar(&flagNoBrowser, "no-browser", false, "disable launching the web-browser post install")

	cmd.MarkFlagsRequiredTogether("docker-username", "docker-password", "docker-email")

	return cmd
}

// envOverride checks if the env exists and is not empty, if that is true
// update the original value to be the value returned from the env environment variable.
// Otherwise, leave the original value alone.
func envOverride(original *string, env string) {
	if v := os.Getenv(env); v != "" {
		*original = v
	}
}
