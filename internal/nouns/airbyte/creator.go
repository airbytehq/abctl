package airbyte

import (
	"context"
	"fmt"

	"github.com/airbytehq/abctl/internal/common"
	"github.com/airbytehq/abctl/internal/helm"
	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/service"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/airbytehq/abctl/internal/trace"
	goHelm "github.com/mittwald/go-helm-client"
	"github.com/pterm/pterm"
	"go.opentelemetry.io/otel/attribute"
)

type Creator struct {
	Chart           string   `help:"Path to chart." xor:"chartver"`
	ChartVersion    string   `help:"Version to install." xor:"chartver"`
	DisableAuth     bool     `help:"Disable auth."`
	DockerEmail     string   `group:"docker" help:"Docker email." env:"ABCTL_LOCAL_INSTALL_DOCKER_EMAIL"`
	DockerPassword  string   `group:"docker" help:"Docker password." env:"ABCTL_LOCAL_INSTALL_DOCKER_PASSWORD"`
	DockerServer    string   `group:"docker" default:"https://index.docker.io/v1/" help:"Docker server." env:"ABCTL_LOCAL_INSTALL_DOCKER_SERVER"`
	DockerUsername  string   `group:"docker" help:"Docker username." env:"ABCTL_LOCAL_INSTALL_DOCKER_USERNAME"`
	Host            []string `help:"HTTP ingress host."`
	InsecureCookies bool     `help:"Allow cookies to be served over HTTP."`
	LowResourceMode bool     `help:"Run Airbyte in low resource mode."`
	NoBrowser       bool     `help:"Disable launching a browser post install."`
	Port            int      `default:"8000" help:"HTTP ingress port."`
	Secret          []string `type:"existingfile" help:"An Airbyte helm chart secret file."`
	Values          string   `type:"existingfile" help:"An Airbyte helm chart values file to configure helm."`
	Volume          []string `help:"Additional volume mounts. Must be in the format <HOST_PATH>:<GUEST_PATH>."`
}

func (c *Creator) Run(ctx context.Context, provider k8s.Provider, newSvcMgrClients service.ManagerClientFactory, telClient telemetry.Client) error {
	ctx, span := trace.NewSpan(ctx, "airbyte create")
	defer span.End()

	extraVolumeMounts, err := k8s.ParseVolumeMounts(c.Volume)
	if err != nil {
		return fmt.Errorf("failed to parse the extra volume mounts: %w", err)
	}

	spinner := &pterm.DefaultSpinner
	spinner, _ = spinner.Start("Starting installation")
	spinner.UpdateText("Checking for Docker installation")

	_, err = dockerInstalled(ctx, telClient)
	if err != nil {
		pterm.Error.Println("Unable to determine if Docker is installed")
		return fmt.Errorf("unable to determine docker installation status: %w", err)
	}

	return telClient.Wrap(ctx, telemetry.Install, func() error {
		spinner.UpdateText(fmt.Sprintf("Checking for existing Kubernetes cluster '%s'", provider.ClusterName))

		cluster, err := provider.Cluster(ctx)
		if err != nil {
			pterm.Error.Printfln("Unable to determine status of any existing '%s' cluster", provider.ClusterName)
			return err
		}

		if cluster.Exists(ctx) {
			pterm.Success.Printfln("Existing cluster '%s' found", provider.ClusterName)
			spinner.UpdateText(fmt.Sprintf("Validating existing cluster '%s'", provider.ClusterName))
			span.SetAttributes(attribute.Bool("cluster_exists", true))

			if provider.Name == k8s.Kind {
				providedPort := c.Port
				c.Port, err = getPort(ctx, provider.ClusterName)
				if err != nil {
					return err
				}
				if providedPort != c.Port {
					pterm.Warning.Printfln("The existing cluster was found to be using port %d, which differs from the provided port %d.\n"+
						"The existing port will be used, as changing ports currently requires the existing installation to be uninstalled first.", c.Port, providedPort)
				}
			}

			pterm.Success.Printfln("Cluster '%s' validation complete", provider.ClusterName)
		} else {
			pterm.Info.Println(fmt.Sprintf("No existing cluster found, cluster '%s' will be created", provider.ClusterName))
			span.SetAttributes(attribute.Bool("cluster_exists", false))

			spinner.UpdateText(fmt.Sprintf("Checking if port %d is available", c.Port))
			if err := portAvailable(ctx, c.Port); err != nil {
				return err
			}
			pterm.Success.Printfln("Port %d appears to be available", c.Port)
			spinner.UpdateText(fmt.Sprintf("Creating cluster '%s'", provider.ClusterName))

			if err := cluster.Create(ctx, c.Port, extraVolumeMounts); err != nil {
				pterm.Error.Printfln("Cluster '%s' could not be created", provider.ClusterName)
				return err
			}
			pterm.Success.Printfln("Cluster '%s' created", provider.ClusterName)
		}

		k8sClient, helmClient, err := newSvcMgrClients(provider.Kubeconfig, provider.Context)
		if err != nil {
			return err
		}

		err = c.setDefaultChartFlags(helmClient)
		if err != nil {
			return fmt.Errorf("failed to set chart defaults: %w", err)
		}

		overrideImages := []string{}

		opts, err := c.installOpts(ctx, telClient.User())
		if err != nil {
			return err
		}

		if opts.EnablePsql17 {
			overrideImages = append(overrideImages, "airbyte/db:"+helm.Psql17AirbyteTag)
		}

		svcMgr, err := service.NewManager(provider,
			service.WithK8sClient(k8sClient),
			service.WithHelmClient(helmClient),
			service.WithPortHTTP(c.Port),
			service.WithTelemetryClient(telClient),
			service.WithSpinner(spinner),
			service.WithDockerClient(dockerAPI),
		)
		if err != nil {
			pterm.Error.Printfln("Failed to initialize 'airbyte' command")
			return fmt.Errorf("unable to initialize airbyte command: %w", err)
		}

		spinner.UpdateText("Pulling images")
		svcMgr.PrepImages(ctx, cluster, opts, overrideImages...)

		if err := svcMgr.Install(ctx, opts); err != nil {
			spinner.Fail("Unable to install Airbyte locally")
			return err
		}

		spinner.Success(
			"Airbyte installation complete.\n" +
				"  A password may be required to login. The password can by found by running\n" +
				"  the command " + pterm.LightBlue("abctl get airbyte --credentials"),
		)
		return nil
	})
}

func (c *Creator) installOpts(ctx context.Context, user string) (*service.InstallOpts, error) {
	ctx, span := trace.NewSpan(ctx, "Creator.installOpts")
	defer span.End()

	span.SetAttributes(attribute.Bool("host", len(c.Host) > 0))

	for _, host := range c.Host {
		if err := validateHostFlag(host); err != nil {
			return nil, err
		}
	}

	supportMinio, err := service.SupportMinio()
	if err != nil {
		return nil, err
	}

	if supportMinio {
		pterm.Warning.Println("Found MinIO physical volume. Consider migrating it to local storage (see project docs)")
	}

	enablePsql17, err := service.EnablePsql17()
	if err != nil {
		return nil, err
	}

	if !enablePsql17 {
		pterm.Warning.Println("Psql 13 detected. Consider upgrading to 17")
	}

	opts := &service.InstallOpts{
		HelmChartVersion: c.ChartVersion,
		AirbyteChartLoc:  c.Chart,
		Secrets:          c.Secret,
		Hosts:            c.Host,
		LocalStorage:     !supportMinio,
		EnablePsql17:     enablePsql17,
		DockerServer:     c.DockerServer,
		DockerUser:       c.DockerUsername,
		DockerPass:       c.DockerPassword,
		DockerEmail:      c.DockerEmail,
		NoBrowser:        c.NoBrowser,
	}

	valuesOpts := helm.ValuesOpts{
		ValuesFile:      c.Values,
		InsecureCookies: c.InsecureCookies,
		LowResourceMode: c.LowResourceMode,
		DisableAuth:     c.DisableAuth,
		LocalStorage:    !supportMinio,
		EnablePsql17:    enablePsql17,
	}

	if opts.DockerAuth() {
		valuesOpts.ImagePullSecret = common.DockerAuthSecretName
	}

	if user != "" {
		valuesOpts.TelemetryUser = user
	}

	valuesYAML, err := helm.BuildAirbyteValues(ctx, valuesOpts, c.ChartVersion)
	if err != nil {
		return nil, err
	}

	opts.HelmValuesYaml = valuesYAML

	return opts, nil
}

func (c *Creator) setDefaultChartFlags(helmClient goHelm.Client) error {
	resolver := helm.NewChartResolver(helmClient)
	resolvedChart, resolvedVersion, err := resolver.ResolveChartReference(c.Chart, c.ChartVersion)
	if err != nil {
		return fmt.Errorf("failed to resolve chart flags: %w", err)
	}

	c.Chart = resolvedChart
	c.ChartVersion = resolvedVersion

	return nil
}