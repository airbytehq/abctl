package airbyte

import (
	"context"

	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/service"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/airbytehq/abctl/internal/trace"
	"github.com/pterm/pterm"
)

type Logger struct {
	Follow    bool   `short:"f" help:"Follow log output."`
	Tail      int    `default:"100" help:"Number of lines to show from the end of the logs."`
	Container string `help:"Specific container to get logs from."`
	Since     string `help:"Only return logs newer than a relative duration like 5s, 2m, or 3h."`
}

func (l *Logger) Run(ctx context.Context, provider k8s.Provider, newSvcMgrClients service.ManagerClientFactory, telClient telemetry.Client) error {
	ctx, span := trace.NewSpan(ctx, "airbyte logs")
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

	k8sClient, _, err := newSvcMgrClients(provider.Kubeconfig, provider.Context)
	if err != nil {
		return err
	}

	logOpts := service.LogOpts{
		Follow:    l.Follow,
		Tail:      l.Tail,
		Container: l.Container,
		Since:     l.Since,
	}

	return service.StreamLogs(ctx, k8sClient, logOpts)
}