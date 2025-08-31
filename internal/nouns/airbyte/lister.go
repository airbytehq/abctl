package airbyte

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/service"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/airbytehq/abctl/internal/trace"
	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

type Lister struct {
	Output string `default:"table" enum:"table,json,yaml" help:"Output format."`
}

func (l *Lister) Run(ctx context.Context, provider k8s.Provider, newSvcMgrClients service.ManagerClientFactory, telClient telemetry.Client) error {
	ctx, span := trace.NewSpan(ctx, "airbyte list")
	defer span.End()

	return telClient.Wrap(ctx, telemetry.Deployments, func() error {
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

		deployments, err := svcMgr.ListDeployments(ctx)
		if err != nil {
			return err
		}

		if len(deployments) == 0 {
			pterm.Println("No Airbyte deployments found")
			return nil
		}

		switch l.Output {
		case "json":
			return json.NewEncoder(os.Stdout).Encode(deployments)
		case "yaml":
			return yaml.NewEncoder(os.Stdout).Encode(deployments)
		default:
			return l.printTable(deployments)
		}
	})
}

func (l *Lister) printTable(deployments []service.DeploymentInfo) error {
	tableData := pterm.TableData{{"Name", "Namespace", "Version", "Status"}}
	
	for _, d := range deployments {
		tableData = append(tableData, []string{
			d.Name,
			d.Namespace,
			d.Version,
			d.Status,
		})
	}

	return pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}