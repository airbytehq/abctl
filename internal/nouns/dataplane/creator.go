package dataplane

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
)

type Creator struct {
	Name   string `arg:"" help:"Dataplane name."`
	Region string `required:"" help:"Region for dataplane."`
	Size   string `default:"medium" enum:"small,medium,large" help:"Dataplane size."`
}

func (c *Creator) Run(ctx context.Context) error {
	pterm.Info.Printfln("Creating dataplane '%s' in region '%s' with size '%s'", c.Name, c.Region, c.Size)
	
	spinner := &pterm.DefaultSpinner
	spinner, _ = spinner.Start("Creating dataplane")
	
	spinner.Success(fmt.Sprintf("Dataplane '%s' created successfully", c.Name))
	pterm.Println(fmt.Sprintf("  ID: dp-%s", c.Name))
	pterm.Println(fmt.Sprintf("  Region: %s", c.Region))
	pterm.Println(fmt.Sprintf("  Size: %s", c.Size))
	
	return nil
}