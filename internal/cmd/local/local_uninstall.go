package local

import (
	"context"
	"fmt"

	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/local"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
)

type UninstallCmd struct {
	Persisted bool `help:"Remove persisted data."`
}

func (u *UninstallCmd) Run(ctx context.Context, provider k8s.Provider, telClient telemetry.Client) error {
	spinner := &pterm.DefaultSpinner

	spinner, _ = spinner.Start("Starting uninstallation")
	spinner.UpdateText("Checking for Docker installation")

	_, err := dockerInstalled(ctx, telClient)
	if err != nil {
		pterm.Error.Println("Unable to determine if Docker is installed")
		return fmt.Errorf("unable to determine docker installation status: %w", err)
	}

	return telClient.Wrap(ctx, telemetry.Uninstall, func() error {
		spinner.UpdateText(fmt.Sprintf("Checking for existing Kubernetes cluster '%s'", provider.ClusterName))

		cluster, err := provider.Cluster(ctx)
		if err != nil {
			pterm.Error.Printfln("Unable to determine if the cluster '%s' exists", provider.ClusterName)
			return err
		}

		// if no cluster exists, there is nothing to do
		if !cluster.Exists(ctx) {
			pterm.Success.Printfln("Cluster '%s' does not exist\nNo additional action required", provider.ClusterName)
			return nil
		}

		pterm.Success.Printfln("Existing cluster '%s' found", provider.ClusterName)

		lc, err := local.New(provider, local.WithTelemetryClient(telClient), local.WithSpinner(spinner))
		if err != nil {
			pterm.Warning.Printfln("Failed to initialize 'local' command\nUninstallation attempt will continue")
			pterm.Debug.Printfln("Initialization of 'local' failed with %s", err.Error())
		} else {
			if err := lc.Uninstall(ctx, local.UninstallOpts{Persisted: u.Persisted}); err != nil {
				pterm.Warning.Printfln("unable to complete uninstall: %s", err.Error())
				pterm.Warning.Println("will still attempt to uninstall the cluster")
			}
		}

		spinner.UpdateText(fmt.Sprintf("Verifying uninstallation status of cluster '%s'", provider.ClusterName))
		if err := cluster.Delete(ctx); err != nil {
			pterm.Error.Printfln(fmt.Sprintf("Uninstallation of cluster '%s' failed", provider.ClusterName))
			return fmt.Errorf("unable to uninstall cluster %s", provider.ClusterName)
		}
		pterm.Success.Printfln(fmt.Sprintf("Uninstallation of cluster '%s' completed successfully", provider.ClusterName))

		spinner.Success("Airbyte uninstallation complete")

		return nil
	})
}
