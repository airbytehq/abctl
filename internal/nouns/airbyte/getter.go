package airbyte

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/airbytehq/abctl/internal/build"
	"github.com/airbytehq/abctl/internal/cmd/images"
	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/service"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/airbytehq/abctl/internal/trace"
	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

type Getter struct {
	Credentials bool   `help:"Get credentials."`
	Status      bool   `help:"Get status (default)."`
	Version     bool   `help:"Get version."`
	Manifest    bool   `help:"Get image manifest."`
	Output      string `default:"table" enum:"table,json,yaml" help:"Output format."`
}

func (g *Getter) Run(ctx context.Context, provider k8s.Provider, newSvcMgrClients service.ManagerClientFactory, telClient telemetry.Client) error {
	switch {
	case g.Credentials:
		return g.getCredentials(ctx, provider, newSvcMgrClients, telClient)
	case g.Version:
		return g.getVersion(ctx)
	case g.Manifest:
		return g.getManifest(ctx, provider, newSvcMgrClients)
	default:
		return g.getStatus(ctx, provider, newSvcMgrClients, telClient)
	}
}

func (g *Getter) getStatus(ctx context.Context, provider k8s.Provider, newSvcMgrClients service.ManagerClientFactory, telClient telemetry.Client) error {
	ctx, span := trace.NewSpan(ctx, "airbyte get status")
	defer span.End()

	cluster, err := provider.Cluster(ctx)
	if err != nil {
		pterm.Error.Printfln("Unable to determine cluster status")
		return err
	}

	if !cluster.Exists(ctx) {
		pterm.Warning.Printfln("Cluster '%s' does not exist", provider.ClusterName)
		return nil
	}

	k8sClient, helmClient, err := newSvcMgrClients(provider.Kubeconfig, provider.Context)
	if err != nil {
		return err
	}

	svcMgr, err := service.NewManager(provider,
		service.WithK8sClient(k8sClient),
		service.WithHelmClient(helmClient),
		service.WithTelemetryClient(telClient),
	)
	if err != nil {
		return fmt.Errorf("unable to initialize service manager: %w", err)
	}

	return telClient.Wrap(ctx, telemetry.Status, func() error {
		status := svcMgr.GetAirbyteStatus(ctx)

		switch g.Output {
		case "json":
			return json.NewEncoder(os.Stdout).Encode(status)
		case "yaml":
			return yaml.NewEncoder(os.Stdout).Encode(status)
		default:
			printStatus(status)
			return nil
		}
	})
}

func (g *Getter) getCredentials(ctx context.Context, provider k8s.Provider, newSvcMgrClients service.ManagerClientFactory, telClient telemetry.Client) error {
	ctx, span := trace.NewSpan(ctx, "airbyte get credentials")
	defer span.End()

	cluster, err := provider.Cluster(ctx)
	if err != nil {
		pterm.Error.Printfln("Unable to determine cluster status")
		return err
	}

	if !cluster.Exists(ctx) {
		pterm.Warning.Printfln("Cluster '%s' does not exist", provider.ClusterName)
		return nil
	}

	k8sClient, helmClient, err := newSvcMgrClients(provider.Kubeconfig, provider.Context)
	if err != nil {
		return err
	}

	svcMgr, err := service.NewManager(provider,
		service.WithK8sClient(k8sClient),
		service.WithHelmClient(helmClient),
		service.WithTelemetryClient(telClient),
	)
	if err != nil {
		return fmt.Errorf("unable to initialize service manager: %w", err)
	}

	return telClient.Wrap(ctx, telemetry.Credentials, func() error {
		creds, err := svcMgr.AuthBasicCredentials(ctx)
		if err != nil {
			return err
		}

		if creds.ClientId == "" && creds.ClientSecret == "" {
			pterm.Println("No credentials found. It is possible basic auth is disabled.")
			return nil
		}

		switch g.Output {
		case "json":
			return json.NewEncoder(os.Stdout).Encode(creds)
		case "yaml":
			return yaml.NewEncoder(os.Stdout).Encode(creds)
		default:
			pterm.Println(fmt.Sprintf("Email: %s", creds.ClientId))
			pterm.Println(fmt.Sprintf("Password: %s", creds.ClientSecret))
			return nil
		}
	})
}

func (g *Getter) getVersion(ctx context.Context) error {
	pterm.Println(fmt.Sprintf("version: %s", build.Version))
	return nil
}

func (g *Getter) getManifest(ctx context.Context, provider k8s.Provider, newSvcMgrClients service.ManagerClientFactory) error {
	manifestCmd := &images.ManifestCmd{
		Chart:        "",
		ChartVersion: "",
		Values:       "",
	}
	return manifestCmd.Run(ctx, newSvcMgrClients)
}

func printStatus(status service.AirbyteStatus) {
	if status.Installed {
		pterm.Println("Airbyte is installed and running")
	} else {
		pterm.Println("Airbyte is not installed")
	}
}