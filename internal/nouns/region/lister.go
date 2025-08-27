package region

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

type Lister struct {
	Available *bool  `help:"Filter by availability."`
	Output    string `default:"table" enum:"table,json,yaml" help:"Output format."`
}

func (l *Lister) Run(ctx context.Context) error {
	regions := []RegionInfo{
		{Name: "us-west-1", Location: "US West (N. California)", Available: true, MaxWorkspaces: 100},
		{Name: "us-west-2", Location: "US West (Oregon)", Available: true, MaxWorkspaces: 100},
		{Name: "us-east-1", Location: "US East (N. Virginia)", Available: true, MaxWorkspaces: 100},
		{Name: "us-east-2", Location: "US East (Ohio)", Available: false, MaxWorkspaces: 100},
		{Name: "eu-west-1", Location: "Europe (Ireland)", Available: true, MaxWorkspaces: 50},
		{Name: "eu-central-1", Location: "Europe (Frankfurt)", Available: true, MaxWorkspaces: 50},
		{Name: "ap-south-1", Location: "Asia Pacific (Mumbai)", Available: true, MaxWorkspaces: 50},
		{Name: "ap-northeast-1", Location: "Asia Pacific (Tokyo)", Available: true, MaxWorkspaces: 50},
	}

	if l.Available != nil {
		var filtered []RegionInfo
		for _, r := range regions {
			if r.Available == *l.Available {
				filtered = append(filtered, r)
			}
		}
		regions = filtered
	}

	if len(regions) == 0 {
		pterm.Println("No regions found")
		return nil
	}

	switch l.Output {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(regions)
	case "yaml":
		return yaml.NewEncoder(os.Stdout).Encode(regions)
	default:
		return l.printTable(regions)
	}
}

func (l *Lister) printTable(regions []RegionInfo) error {
	tableData := pterm.TableData{{"Name", "Location", "Available", "Max Workspaces"}}
	
	for _, r := range regions {
		available := "Yes"
		if !r.Available {
			available = "No"
		}
		tableData = append(tableData, []string{
			r.Name,
			r.Location,
			available,
			fmt.Sprintf("%d", r.MaxWorkspaces),
		})
	}

	return pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}