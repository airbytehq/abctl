package dataplane

import (
	"context"
	"encoding/json"
	"os"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

type Lister struct {
	Region string `help:"Filter by region."`
	Output string `default:"table" enum:"table,json,yaml" help:"Output format."`
}

func (l *Lister) Run(ctx context.Context) error {
	dataplanes := []DataplaneInfo{
		{Name: "prod-dataplane", ID: "dp-prod", Region: "us-west-2", Status: "active"},
		{Name: "dev-dataplane", ID: "dp-dev", Region: "us-east-1", Status: "active"},
	}

	if l.Region != "" {
		var filtered []DataplaneInfo
		for _, dp := range dataplanes {
			if dp.Region == l.Region {
				filtered = append(filtered, dp)
			}
		}
		dataplanes = filtered
	}

	if len(dataplanes) == 0 {
		pterm.Println("No dataplanes found")
		return nil
	}

	switch l.Output {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(dataplanes)
	case "yaml":
		return yaml.NewEncoder(os.Stdout).Encode(dataplanes)
	default:
		return l.printTable(dataplanes)
	}
}

func (l *Lister) printTable(dataplanes []DataplaneInfo) error {
	tableData := pterm.TableData{{"Name", "ID", "Region", "Status"}}
	
	for _, dp := range dataplanes {
		tableData = append(tableData, []string{
			dp.Name,
			dp.ID,
			dp.Region,
			dp.Status,
		})
	}

	return pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}