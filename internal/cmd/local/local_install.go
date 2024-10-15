package local

import (
	"context"
	"fmt"
	"strings"

	"github.com/airbytehq/abctl/internal/cmd/local/helm"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/local"
	"github.com/airbytehq/abctl/internal/common"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/airbytehq/abctl/internal/trace"
	"github.com/pterm/pterm"
	"go.opentelemetry.io/otel/attribute"
)

type InstallCmd struct {
	Chart           string   `help:"Path to chart." xor:"chartver"`
	ChartVersion    string   `help:"Version to install." xor:"chartver"`
	DockerEmail     string   `group:"docker" help:"Docker email." env:"ABCTL_LOCAL_INSTALL_DOCKER_EMAIL"`
	DockerPassword  string   `group:"docker" help:"Docker password." env:"ABCTL_LOCAL_INSTALL_DOCKER_PASSWORD"`
	DockerServer    string   `group:"docker" default:"https://index.docker.io/v1/" help:"Docker server." env:"ABCTL_LOCAL_INSTALL_DOCKER_SERVER"`
	DockerUsername  string   `group:"docker" help:"Docker username." env:"ABCTL_LOCAL_INSTALL_DOCKER_USERNAME"`
	Host            []string `help:"HTTP ingress host."`
	InsecureCookies bool     `help:"Allow cookies to be served over HTTP."`
	LowResourceMode bool     `help:"Run Airbyte in low resource mode."`
	Migrate         bool     `help:"Migrate data from a previous Docker Compose Airbyte installation."`
	NoBrowser       bool     `help:"Disable launching a browser post install."`
	Port            int      `default:"8000" help:"HTTP ingress port."`
	Secret          []string `type:"existingfile" help:"An Airbyte helm chart secret file."`
	Values          string   `type:"existingfile" help:"An Airbyte helm chart values file to configure helm."`
	Volume          []string `help:"Additional volume mounts. Must be in the format <HOST_PATH>:<GUEST_PATH>."`
}

func (i *InstallCmd) InstallOpts(ctx context.Context, user string) (*local.InstallOpts, error) {
	ctx, span := trace.NewSpan(ctx, "InstallCmd.InstallOpts")
	defer span.End()

	extraVolumeMounts, err := parseVolumeMounts(i.Volume)
	if err != nil {
		return nil, err
	}

	span.SetAttributes(attribute.Bool("host", len(i.Host) > 0))

	for _, host := range i.Host {
		if err := validateHostFlag(host); err != nil {
			return nil, err
		}
	}

	opts := &local.InstallOpts{
		HelmChartVersion:  i.ChartVersion,
		AirbyteChartLoc:   helm.LocateLatestAirbyteChart(i.ChartVersion, i.Chart),
		Secrets:           i.Secret,
		Migrate:           i.Migrate,
		Hosts:             i.Host,
		ExtraVolumeMounts: extraVolumeMounts,
		DockerServer:      i.DockerServer,
		DockerUser:        i.DockerUsername,
		DockerPass:        i.DockerPassword,
		DockerEmail:       i.DockerEmail,
		NoBrowser:         i.NoBrowser,
	}

	valuesOpts := helm.ValuesOpts{
		ValuesFile:      i.Values,
		InsecureCookies: i.InsecureCookies,
		LowResourceMode: i.LowResourceMode,
	}

	if opts.DockerAuth() {
		valuesOpts.ImagePullSecret = common.DockerAuthSecretName
	}

	// only override the empty telUser if the tel.User returns a non-nil (uuid.Nil) value.
	if user != "" {
		valuesOpts.TelemetryUser = user
	}

	valuesYAML, err := helm.BuildAirbyteValues(ctx, valuesOpts)
	if err != nil {
		return nil, err
	}
	opts.HelmValuesYaml = valuesYAML

	return opts, nil
}

func (i *InstallCmd) Run(ctx context.Context, provider k8s.Provider, telClient telemetry.Client) error {
	ctx, span := trace.NewSpan(ctx, "local install")
	defer span.End()

	spinner := &pterm.DefaultSpinner
	spinner, _ = spinner.Start("Starting installation")
	spinner.UpdateText("Checking for Docker installation")

	_, err := dockerInstalled(ctx, telClient)
	if err != nil {
		pterm.Error.Println("Unable to determine if Docker is installed")
		return fmt.Errorf("unable to determine docker installation status: %w", err)
	}

	opts, err := i.InstallOpts(ctx, telClient.User())
	if err != nil {
		return err
	}

	return telClient.Wrap(ctx, telemetry.Install, func() error {
		spinner.UpdateText(fmt.Sprintf("Checking for existing Kubernetes cluster '%s'", provider.ClusterName))

		cluster, err := provider.Cluster(ctx)
		if err != nil {
			pterm.Error.Printfln("Unable to determine status of any existing '%s' cluster", provider.ClusterName)
			return err
		}

		if cluster.Exists(ctx) {
			// existing cluster, validate it
			pterm.Success.Printfln("Existing cluster '%s' found", provider.ClusterName)
			spinner.UpdateText(fmt.Sprintf("Validating existing cluster '%s'", provider.ClusterName))

			// only for kind do we need to check the existing port
			if provider.Name == k8s.Kind {
				providedPort := i.Port
				i.Port, err = getPort(ctx, provider.ClusterName)
				if err != nil {
					return err
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

			spinner.UpdateText(fmt.Sprintf("Checking if port %d is available", i.Port))
			if err := portAvailable(ctx, i.Port); err != nil {
				return err
			}
			pterm.Success.Printfln("Port %d appears to be available", i.Port)
			spinner.UpdateText(fmt.Sprintf("Creating cluster '%s'", provider.ClusterName))

			if err := cluster.Create(ctx, i.Port, opts.ExtraVolumeMounts); err != nil {
				pterm.Error.Printfln("Cluster '%s' could not be created", provider.ClusterName)
				return err
			}
			pterm.Success.Printfln("Cluster '%s' created", provider.ClusterName)
		}

		lc, err := local.New(provider,
			local.WithPortHTTP(i.Port),
			local.WithTelemetryClient(telClient),
			local.WithSpinner(spinner),
			local.WithDockerClient(dockerClient),
		)
		if err != nil {
			pterm.Error.Printfln("Failed to initialize 'local' command")
			return fmt.Errorf("unable to initialize local command: %w", err)
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
	if len(specs) == 0 {
		return nil, nil
	}

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
