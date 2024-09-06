package local

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
)

// Status handles the status of local Airbyte.
func (c *Command) Status(_ context.Context) error {
	charts := []string{airbyteChartRelease, nginxChartRelease}
	for _, name := range charts {
		c.spinner.UpdateText(fmt.Sprintf("Verifying %s Helm Chart installation status", name))

		rel, err := c.helm.GetRelease(name)
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

	pterm.Info.Println(fmt.Sprintf("Airbyte should be accessible via http://localhost:%d", c.portHTTP))

	return nil
}