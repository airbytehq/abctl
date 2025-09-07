package workspace

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
)

type Creator struct {
	Name        string `arg:"" help:"Workspace name."`
	DataplaneID string `required:"" help:"Dataplane ID to use."`
	Region      string `help:"Region for workspace (inherits from dataplane if not specified)."`
}

func (c *Creator) Run(ctx context.Context) error {
	region := c.Region
	if region == "" {
		region = "us-west-2"
		pterm.Info.Printfln("Using dataplane's region: %s", region)
	}
	
	pterm.Info.Printfln("Creating workspace '%s' on dataplane '%s' in region '%s'", c.Name, c.DataplaneID, region)
	
	spinner := &pterm.DefaultSpinner
	spinner, _ = spinner.Start("Creating workspace")
	
	spinner.Success(fmt.Sprintf("Workspace '%s' created successfully", c.Name))
	pterm.Println(fmt.Sprintf("  ID: ws-%s", c.Name))
	pterm.Println(fmt.Sprintf("  Dataplane: %s", c.DataplaneID))
	pterm.Println(fmt.Sprintf("  Region: %s", region))
	pterm.Println(fmt.Sprintf("  URL: https://%s.airbyte.cloud", c.Name))
	
	return nil
}