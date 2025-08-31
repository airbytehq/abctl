package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

type Getter struct {
	Name   string `arg:"" help:"Workspace name."`
	Output string `default:"table" enum:"table,json,yaml" help:"Output format."`
}

func (g *Getter) Run(ctx context.Context) error {
	workspace := WorkspaceInfo{
		Name:        g.Name,
		ID:          fmt.Sprintf("ws-%s", g.Name),
		DataplaneID: "dp-prod",
		Region:      "us-west-2",
		Status:      "active",
		CreatedAt:   "2024-01-01T12:00:00Z",
	}

	switch g.Output {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(workspace)
	case "yaml":
		return yaml.NewEncoder(os.Stdout).Encode(workspace)
	default:
		pterm.Println(fmt.Sprintf("Name: %s", workspace.Name))
		pterm.Println(fmt.Sprintf("ID: %s", workspace.ID))
		pterm.Println(fmt.Sprintf("Dataplane: %s", workspace.DataplaneID))
		pterm.Println(fmt.Sprintf("Region: %s", workspace.Region))
		pterm.Println(fmt.Sprintf("Status: %s", workspace.Status))
		pterm.Println(fmt.Sprintf("Created: %s", workspace.CreatedAt))
		return nil
	}
}

type WorkspaceInfo struct {
	Name        string `json:"name" yaml:"name"`
	ID          string `json:"id" yaml:"id"`
	DataplaneID string `json:"dataplane_id" yaml:"dataplane_id"`
	Region      string `json:"region" yaml:"region"`
	Status      string `json:"status" yaml:"status"`
	CreatedAt   string `json:"created_at" yaml:"created_at"`
}