package local

import (
	"context"
	"fmt"
	"strings"

	"github.com/airbytehq/abctl/internal/cmd/local/docker"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/local"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
)

type InstallCmd struct {
	ChartVersion    string   `default:"latest" help:"Version to install."`
	DockerEmail     string   `group:"docker" help:"Docker email." env:"ABCTL_LOCAL_INSTALL_DOCKER_EMAIL"`
	DockerPassword  string   `group:"docker" help:"Docker password." env:"ABCTL_LOCAL_INSTALL_DOCKER_PASSWORD"`
	DockerServer    string   `group:"docker" default:"https://index.docker.io/v1/" help:"Docker server." env:"ABCTL_LOCAL_INSTALL_DOCKER_SERVER"`
	DockerUsername  string   `group:"docker" help:"Docker username." env:"ABCTL_LOCAL_INSTALL_DOCKER_USERNAME"`
	Host            string   `default:"localhost" help:"HTTP ingress host."`
	InsecureCookies bool     `help:"Allow cookies to be served over HTTP."`
	LowResourceMode bool     `help:"Run Airbyte in low resource mode."`
	Migrate         bool     `help:"Migrate data from a previous Docker Compose Airbyte installation."`
	NoBrowser       bool     `help:"Disable launching a browser post install."`
	Port            int      `default:"8000" help:"HTTP ingress port."`
	Secret          []string `type:"existingfile" help:"An Airbyte helm chart secret file."`
	Values          string   `type:"existingfile" help:"An Airbyte helm chart values file to configure helm."`
	Volume          []string `help:"Additional volume mounts. Must be in the format <HOST_PATH>:<GUEST_PATH>."`
}

func (i *InstallCmd) Run(ctx context.Context, provider k8s.Provider, telClient telemetry.Client) error {
	spinner := &pterm.DefaultSpinner
	spinner, _ = spinner.Start("Starting installation")
	spinner.UpdateText("Checking for Docker installation")

	dockerVersion, err := dockerInstalled(ctx)
	if err != nil {
		pterm.Error.Println("Unable to determine if Docker is installed")
		return fmt.Errorf("unable to determine docker installation status: %w", err)
	}

	telClient.Attr("docker_version", dockerVersion.Version)
	telClient.Attr("docker_arch", dockerVersion.Arch)
	telClient.Attr("docker_platform", dockerVersion.Platform)

	spinner.UpdateText(fmt.Sprintf("Checking if port %d is available", i.Port))
	if err := portAvailable(ctx, i.Port); err != nil {
		return fmt.Errorf("port %d is not available: %w", i.Port, err)
	}

	return telClient.Wrap(ctx, telemetry.Install, func() error {
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
					dockerClient, err = docker.New(ctx)
					if err != nil {
						pterm.Error.Printfln("Unable to connect to Docker daemon")
						return fmt.Errorf("unable to connect to docker: %w", err)
					}
				}

				providedPort := i.Port
				i.Port, err = dockerClient.Port(ctx, fmt.Sprintf("%s-control-plane", provider.ClusterName))
				if err != nil {
					pterm.Warning.Printfln("Unable to determine which port the existing cluster was configured to use.\n" +
						"Installation will continue but may ultimately fail, in which case it will be necessarily to uninstall first.")
					// since we can't verify the port is correct, push forward with the provided port
					i.Port = providedPort
				}
				if providedPort != i.Port {
					pterm.Warning.Printfln("The existing cluster was found to be using port %d, which differs from the provided port %d.\n"+
						"The existing port will be used, as changing ports currently requires the existing installation to be uninstalled first.", i.Port, providedPort)
				}
			}

			pterm.Success.Printfln("Cluster '%s' validation complete", provider.ClusterName)
		} else {
			// no existing cluster, need to create one
			pterm.Info.Println(fmt.Sprintf("No existing cluster found, cluster '%s' will be created", provider.ClusterName))
			spinner.UpdateText(fmt.Sprintf("Creating cluster '%s'", provider.ClusterName))

			extraVolumeMounts, err := parseVolumeMounts(i.Volume)
			if err != nil {
				return err
			}

			if err := cluster.Create(i.Port, extraVolumeMounts); err != nil {
				pterm.Error.Printfln("Cluster '%s' could not be created", provider.ClusterName)
				return err
			}
			pterm.Success.Printfln("Cluster '%s' created", provider.ClusterName)
		}

		lc, err := local.New(provider,
			local.WithPortHTTP(i.Port),
			local.WithTelemetryClient(telClient),
			local.WithSpinner(spinner),
		)
		if err != nil {
			pterm.Error.Printfln("Failed to initialize 'local' command")
			return fmt.Errorf("unable to initialize local command: %w", err)
		}

		opts := local.InstallOpts{
			HelmChartVersion: i.ChartVersion,
			ValuesFile:       i.Values,
			Secrets:          i.Secret,
			Migrate:          i.Migrate,
			Docker:           dockerClient,
			Host:             i.Host,

			DockerServer: i.DockerServer,
			DockerUser:   i.DockerUsername,
			DockerPass:   i.DockerPassword,
			DockerEmail:  i.DockerEmail,

			NoBrowser:       i.NoBrowser,
			LowResourceMode: i.LowResourceMode,
			InsecureCookies: i.InsecureCookies,
		}

		if opts.HelmChartVersion == "latest" {
			opts.HelmChartVersion = ""
		}

		if err := lc.Install(ctx, opts); err != nil {
			spinner.Fail("Unable to install Airbyte locally")
			return err
		}

		spinner.Success(
			"Airbyte installation complete.\n" +
				"  A password may be required to login. The password can by found by running\n" +
				"  the command " + pterm.LightBlue("abctl local credentials"),
		)
		return nil
	})
}

func parseVolumeMounts(specs []string) ([]k8s.ExtraVolumeMount, error) {
	mounts := make([]k8s.ExtraVolumeMount, len(specs))

	for i, spec := range specs {
		parts := strings.Split(spec, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("volume %s is not a valid volume spec, must be <HOST_PATH>:<GUEST_PATH>", spec)
		}
		mounts[i] = k8s.ExtraVolumeMount{
			HostPath:      parts[0],
			ContainerPath: parts[1],
		}
	}

	return mounts, nil
}
