package service

import (
	"context"
	"fmt"

	"github.com/airbytehq/abctl/internal/common"
	"github.com/pterm/pterm"
	"go.opencensus.io/trace"
)

// Status handles the status of local Airbyte.
func (m *Manager) Status(ctx context.Context) error {
	_, span := trace.StartSpan(ctx, "command.Status")
	defer span.End()

	charts := []string{common.AirbyteChartRelease, common.NginxChartRelease}
	for _, name := range charts {
		m.spinner.UpdateText(fmt.Sprintf("Verifying %s Helm Chart installation status", name))

		rel, err := m.helm.GetRelease(name)
		if err != nil {
			pterm.Warning.Println("Unable to fetch airbyte release")
			pterm.Debug.Printfln("unable to fetch airbyte release: %s", err)
			continue
		}

		pterm.Info.Println(fmt.Sprintf(
			"Found helm chart '%s'\n  Status: %s\n  Chart Version: %s\n  App Version: %s",
			name, rel.Info.Status.String(), rel.Chart.Metadata.Version, rel.Chart.Metadata.AppVersion,
		))
	}

	pterm.Info.Println(fmt.Sprintf("Airbyte should be accessible via http://localhost:%d", m.portHTTP))

	return nil
}
