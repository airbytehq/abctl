package airbyte

import (
	"context"
	"fmt"

	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/service"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/airbytehq/abctl/internal/trace"
	"github.com/pterm/pterm"
)

type Editor struct {
	Port            *int     `help:"Change HTTP ingress port."`
	Host            []string `help:"Change HTTP ingress hosts."`
	DisableAuth     *bool    `help:"Enable/disable authentication."`
	LowResourceMode *bool    `help:"Enable/disable low resource mode."`
	Values          string   `type:"existingfile" help:"An Airbyte helm chart values file to configure helm."`
}

func (e *Editor) Run(ctx context.Context, provider k8s.Provider, newSvcMgrClients service.ManagerClientFactory, telClient telemetry.Client) error {
	ctx, span := trace.NewSpan(ctx, "airbyte edit")
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

	spinner := &pterm.DefaultSpinner
	spinner, _ = spinner.Start("Updating Airbyte configuration")

	svcMgr, err := service.NewManager(provider,
		service.WithK8sClient(k8sClient),
		service.WithHelmClient(helmClient),
		service.WithTelemetryClient(telClient),
		service.WithSpinner(spinner),
	)
	if err != nil {
		spinner.Fail("Unable to initialize service manager")
		return fmt.Errorf("unable to initialize service manager: %w", err)
	}

	updateOpts := service.UpdateOpts{
		ValuesFile: e.Values,
	}

	if e.Port != nil {
		updateOpts.Port = *e.Port
	}

	if len(e.Host) > 0 {
		for _, host := range e.Host {
			if err := validateHostFlag(host); err != nil {
				spinner.Fail("Invalid host configuration")
				return err
			}
		}
		updateOpts.Hosts = e.Host
	}

	if e.DisableAuth != nil {
		updateOpts.DisableAuth = e.DisableAuth
	}

	if e.LowResourceMode != nil {
		updateOpts.LowResourceMode = e.LowResourceMode
	}

	if err := svcMgr.Update(ctx, updateOpts); err != nil {
		spinner.Fail("Unable to update Airbyte configuration")
		return err
	}

	spinner.Success("Airbyte configuration updated successfully")
	return nil
}