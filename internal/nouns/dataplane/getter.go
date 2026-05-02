package dataplane

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

type Getter struct {
	Name   string `arg:"" help:"Dataplane name."`
	Output string `default:"table" enum:"table,json,yaml" help:"Output format."`
}

func (g *Getter) Run(ctx context.Context) error {
	dataplane := DataplaneInfo{
		Name:   g.Name,
		ID:     fmt.Sprintf("dp-%s", g.Name),
		Region: "us-west-2",
		Status: "active",
	}

	switch g.Output {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(dataplane)
	case "yaml":
		return yaml.NewEncoder(os.Stdout).Encode(dataplane)
	default:
		pterm.Println(fmt.Sprintf("Name: %s", dataplane.Name))
		pterm.Println(fmt.Sprintf("ID: %s", dataplane.ID))
		pterm.Println(fmt.Sprintf("Region: %s", dataplane.Region))
		pterm.Println(fmt.Sprintf("Status: %s", dataplane.Status))
		return nil
	}
}

type DataplaneInfo struct {
	Name   string `json:"name" yaml:"name"`
	ID     string `json:"id" yaml:"id"`
	Region string `json:"region" yaml:"region"`
	Status string `json:"status" yaml:"status"`
}