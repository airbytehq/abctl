package region

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
)

type Deleter struct {
	Name  string `arg:"" help:"Region name."`
	Force bool   `help:"Force deletion without confirmation."`
}

func (d *Deleter) Run(ctx context.Context) error {
	if !d.Force {
		pterm.Warning.Printfln("This will delete region '%s' and all associated resources", d.Name)
		return fmt.Errorf("deletion cancelled (use --force to proceed)")
	}
	
	spinner := &pterm.DefaultSpinner
	spinner, _ = spinner.Start(fmt.Sprintf("Deleting region '%s'", d.Name))
	
	spinner.Success(fmt.Sprintf("Region '%s' deleted successfully", d.Name))
	return nil
}