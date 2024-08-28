package local

import (
	"context"
	"fmt"

	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
)

type DeploymentsCmd struct {
	Restart string `help:"Deployment to restart."`
}

func (d *DeploymentsCmd) Run(ctx context.Context, provider k8s.Provider, telClient telemetry.Client) error {
	spinner := &pterm.DefaultSpinner
	if err := checkDocker(ctx, telClient, spinner); err != nil {
		return err
	}

	return telClient.Wrap(ctx, telemetry.Restart, func() error {
		return d.deployments(ctx, provider, telClient, spinner)
	})
}

func (d *DeploymentsCmd) deployments(ctx context.Context, provider k8s.Provider, telClient telemetry.Client, spinner *pterm.SpinnerPrinter) error {
	k8sClient, err := defaultK8s(provider.Kubeconfig, provider.Context)
	if err != nil {
		pterm.Error.Println("No existing cluster found")
		return nil
	}

	if d.Restart == "" {
		spinner.UpdateText("Fetching deployments")
		svcs, err := k8sClient.DeploymentList(ctx, airbyteNamespace)
		if err != nil {
			pterm.Error.Println("Unable to list deployments")
			return fmt.Errorf("unable to list deployments: %w", err)
		}

		if len(svcs.Items) == 0 {
			pterm.Info.Println("No deployments found")
			return nil
		}
		output := "Found the following deployments:"
		for _, svc := range svcs.Items {
			output += "\n  " + svc.Name
		}
		pterm.Info.Println(output)

		return nil
	}

	spinner.UpdateText(fmt.Sprintf("Restarting deployment %s", d.Restart))
	if err := k8sClient.DeploymentRestart(ctx, airbyteNamespace, d.Restart); err != nil {
		pterm.Error.Println(fmt.Sprintf("Unable to restart airbyte deployment %s", d.Restart))
		return fmt.Errorf("unable to restart airbyte deployment: %w", err)
	}

	return nil
}
