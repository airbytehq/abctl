package region

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
)

type Creator struct {
	Name          string `arg:"" help:"Region name."`
	Location      string `required:"" help:"Region location."`
	MaxWorkspaces int    `default:"100" help:"Maximum number of workspaces."`
}

func (c *Creator) Run(ctx context.Context) error {
	pterm.Info.Printfln("Creating region '%s' at location '%s'", c.Name, c.Location)
	
	spinner := &pterm.DefaultSpinner
	spinner, _ = spinner.Start("Creating region")
	
	spinner.Success(fmt.Sprintf("Region '%s' created successfully", c.Name))
	pterm.Println(fmt.Sprintf("  Location: %s", c.Location))
	pterm.Println(fmt.Sprintf("  Max Workspaces: %d", c.MaxWorkspaces))
	
	return nil
}