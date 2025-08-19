package region

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

type Getter struct {
	Name   string `arg:"" help:"Region name."`
	Output string `default:"table" enum:"table,json,yaml" help:"Output format."`
}

func (g *Getter) Run(ctx context.Context) error {
	region := RegionInfo{
		Name:         g.Name,
		Location:     getLocation(g.Name),
		Available:    true,
		MaxWorkspaces: 100,
	}

	switch g.Output {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(region)
	case "yaml":
		return yaml.NewEncoder(os.Stdout).Encode(region)
	default:
		pterm.Println(fmt.Sprintf("Name: %s", region.Name))
		pterm.Println(fmt.Sprintf("Location: %s", region.Location))
		pterm.Println(fmt.Sprintf("Available: %v", region.Available))
		pterm.Println(fmt.Sprintf("Max Workspaces: %d", region.MaxWorkspaces))
		return nil
	}
}

type RegionInfo struct {
	Name          string `json:"name" yaml:"name"`
	Location      string `json:"location" yaml:"location"`
	Available     bool   `json:"available" yaml:"available"`
	MaxWorkspaces int    `json:"max_workspaces" yaml:"max_workspaces"`
}

func getLocation(name string) string {
	locations := map[string]string{
		"us-west-1":    "US West (N. California)",
		"us-west-2":    "US West (Oregon)",
		"us-east-1":    "US East (N. Virginia)",
		"us-east-2":    "US East (Ohio)",
		"eu-west-1":    "Europe (Ireland)",
		"eu-central-1": "Europe (Frankfurt)",
		"ap-south-1":   "Asia Pacific (Mumbai)",
		"ap-northeast-1": "Asia Pacific (Tokyo)",
	}
	
	if loc, ok := locations[name]; ok {
		return loc
	}
	return "Unknown"
}