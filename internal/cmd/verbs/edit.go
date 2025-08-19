package verbs

import (
	"github.com/airbytehq/abctl/internal/nouns/airbyte"
	"github.com/airbytehq/abctl/internal/nouns/dataplane"
	"github.com/airbytehq/abctl/internal/nouns/region"
	"github.com/airbytehq/abctl/internal/nouns/workspace"
)

type EditCmd struct {
	Airbyte    airbyte.Editor   `cmd:"" help:"Edit Airbyte configuration."`
	Airbytes   airbyte.Editor   `cmd:"airbytes" aliases:"airbytes" help:"Edit Airbyte configuration (plural)."`
	Dataplane  dataplane.Editor `cmd:"" help:"Edit dataplane configuration."`
	Dataplanes dataplane.Editor `cmd:"dataplanes" aliases:"dataplanes" help:"Edit dataplane configuration (plural)."`
	Region     region.Editor    `cmd:"" help:"Edit region configuration."`
	Regions    region.Editor    `cmd:"regions" aliases:"regions" help:"Edit region configuration (plural)."`
	Workspace  workspace.Editor `cmd:"" help:"Edit workspace configuration."`
	Workspaces workspace.Editor `cmd:"workspaces" aliases:"workspaces" help:"Edit workspace configuration (plural)."`
}