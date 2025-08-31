package verbs

import (
	"github.com/airbytehq/abctl/internal/nouns/airbyte"
	"github.com/airbytehq/abctl/internal/nouns/dataplane"
	"github.com/airbytehq/abctl/internal/nouns/region"
	"github.com/airbytehq/abctl/internal/nouns/workspace"
)

type ListCmd struct {
	Airbyte    airbyte.Lister   `cmd:"" help:"List Airbyte deployments."`
	Airbytes   airbyte.Lister   `cmd:"airbytes" aliases:"airbytes" help:"List Airbyte deployments (plural)."`
	Dataplane  dataplane.Lister `cmd:"" help:"List dataplanes."`
	Dataplanes dataplane.Lister `cmd:"dataplanes" aliases:"dataplanes" help:"List dataplanes (plural)."`
	Region     region.Lister    `cmd:"" help:"List regions."`
	Regions    region.Lister    `cmd:"regions" aliases:"regions" help:"List regions (plural)."`
	Workspace  workspace.Lister `cmd:"" help:"List workspaces."`
	Workspaces workspace.Lister `cmd:"workspaces" aliases:"workspaces" help:"List workspaces (plural)."`
}