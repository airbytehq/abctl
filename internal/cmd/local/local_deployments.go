package local

import (
	"context"
	"fmt"

	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/local"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
	"go.opencensus.io/trace"
)

type DeploymentsCmd struct {
	Restart string `help:"Deployment to restart."`
}

func (d *DeploymentsCmd) Run(ctx context.Context, telClient telemetry.Client, provider k8s.Provider) error {
	ctx, span := trace.StartSpan(ctx, "local deployments")
	defer span.End()

	k8sClient, err := local.DefaultK8s(provider.Kubeconfig, provider.Context)
	if err != nil {
		return err
	}

	spinner := &pterm.DefaultSpinner
	if err := checkDocker(ctx, telClient, spinner); err != nil {
		return err
	}

	return telClient.Wrap(ctx, telemetry.Deployments, func() error {
		return d.deployments(ctx, k8sClient, spinner)
	})
}

func (d *DeploymentsCmd) deployments(ctx context.Context, k8sClient k8s.Client, spinner *pterm.SpinnerPrinter) error {
	if d.Restart == "" {
		spinner.UpdateText("Fetching deployments")
		deployments, err := k8sClient.DeploymentList(ctx, airbyteNamespace)
		if err != nil {
			pterm.Error.Println("Unable to list deployments")
			return fmt.Errorf("unable to list deployments: %w", err)
		}

		if len(deployments.Items) == 0 {
			pterm.Info.Println("No deployments found")
			return nil
		}
		output := "Found the following deployments:"
		for _, deployment := range deployments.Items {
			output += "\n  " + deployment.Name
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
