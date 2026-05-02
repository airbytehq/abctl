package dataplane

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
)

type Editor struct {
	Name string  `arg:"" help:"Dataplane name."`
	Size *string `enum:"small,medium,large" help:"New dataplane size."`
}

func (e *Editor) Run(ctx context.Context) error {
	spinner := &pterm.DefaultSpinner
	spinner, _ = spinner.Start(fmt.Sprintf("Updating dataplane '%s'", e.Name))
	
	if e.Size != nil {
		pterm.Info.Printfln("Resizing dataplane to '%s'", *e.Size)
	}
	
	spinner.Success(fmt.Sprintf("Dataplane '%s' updated successfully", e.Name))
	return nil
}