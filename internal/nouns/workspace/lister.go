package workspace

import (
	"context"
	"encoding/json"
	"os"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

type Lister struct {
	Region      string `help:"Filter by region."`
	DataplaneID string `help:"Filter by dataplane ID."`
	Output      string `default:"table" enum:"table,json,yaml" help:"Output format."`
}

func (l *Lister) Run(ctx context.Context) error {
	workspaces := []WorkspaceInfo{
		{Name: "production", ID: "ws-production", DataplaneID: "dp-prod", Region: "us-west-2", Status: "active", CreatedAt: "2024-01-01T12:00:00Z"},
		{Name: "staging", ID: "ws-staging", DataplaneID: "dp-prod", Region: "us-west-2", Status: "active", CreatedAt: "2024-01-02T12:00:00Z"},
		{Name: "development", ID: "ws-development", DataplaneID: "dp-dev", Region: "us-east-1", Status: "active", CreatedAt: "2024-01-03T12:00:00Z"},
	}

	if l.Region != "" {
		var filtered []WorkspaceInfo
		for _, ws := range workspaces {
			if ws.Region == l.Region {
				filtered = append(filtered, ws)
			}
		}
		workspaces = filtered
	}

	if l.DataplaneID != "" {
		var filtered []WorkspaceInfo
		for _, ws := range workspaces {
			if ws.DataplaneID == l.DataplaneID {
				filtered = append(filtered, ws)
			}
		}
		workspaces = filtered
	}

	if len(workspaces) == 0 {
		pterm.Println("No workspaces found")
		return nil
	}

	switch l.Output {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(workspaces)
	case "yaml":
		return yaml.NewEncoder(os.Stdout).Encode(workspaces)
	default:
		return l.printTable(workspaces)
	}
}

func (l *Lister) printTable(workspaces []WorkspaceInfo) error {
	tableData := pterm.TableData{{"Name", "ID", "Dataplane", "Region", "Status"}}
	
	for _, ws := range workspaces {
		tableData = append(tableData, []string{
			ws.Name,
			ws.ID,
			ws.DataplaneID,
			ws.Region,
			ws.Status,
		})
	}

	return pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}