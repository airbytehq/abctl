package airbyte

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/service"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/airbytehq/abctl/internal/trace"
	"github.com/pterm/pterm"
)

type Deleter struct {
	Force           bool `help:"Force deletion without confirmation."`
	PersistVolumes  bool `help:"Preserve persisted data."`
}

func (d *Deleter) Run(ctx context.Context, provider k8s.Provider, newSvcMgrClients service.ManagerClientFactory, telClient telemetry.Client) error {
	ctx, span := trace.NewSpan(ctx, "airbyte delete")
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

	if !d.Force {
		ok, err := getConfirmation("❗ This will remove the Airbyte installation and all of its data.")
		if err != nil {
			return err
		}
		if !ok {
			pterm.Println("Deletion cancelled")
			return nil
		}
	}

	k8sClient, helmClient, err := newSvcMgrClients(provider.Kubeconfig, provider.Context)
	if err != nil {
		return err
	}

	spinner := &pterm.DefaultSpinner
	spinner, _ = spinner.Start("Uninstalling Airbyte")

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

	return telClient.Wrap(ctx, telemetry.Uninstall, func() error {
		if err := svcMgr.Uninstall(ctx, service.UninstallOpts{Persisted: d.PersistVolumes}); err != nil {
			spinner.Fail("Unable to uninstall Airbyte")
			return err
		}

		spinner.Success("Airbyte uninstalled")
		if d.PersistVolumes {
			pterm.Println("  Data has been preserved and will be reused if Airbyte is reinstalled.")
		}
		return nil
	})
}

func getConfirmation(msg string) (bool, error) {
	if msg != "" {
		pterm.Println(msg)
	}
	pterm.Print("  Enter 'yes' to continue: ")

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("unable to read answer: %w", err)
	}

	answer = strings.TrimSpace(answer)
	if strings.EqualFold(answer, "yes") {
		return true, nil
	}

	return false, nil
}