package verbs

import (
	"github.com/airbytehq/abctl/internal/nouns/airbyte"
	"github.com/airbytehq/abctl/internal/nouns/dataplane"
	"github.com/airbytehq/abctl/internal/nouns/region"
	"github.com/airbytehq/abctl/internal/nouns/workspace"
)

type CreateCmd struct {
	Airbyte    airbyte.Creator   `cmd:"" help:"Create local Airbyte installation."`
	Airbytes   airbyte.Creator   `cmd:"airbytes" aliases:"airbytes" help:"Create local Airbyte installation (plural)."`
	Dataplane  dataplane.Creator `cmd:"" help:"Create dataplane."`
	Dataplanes dataplane.Creator `cmd:"dataplanes" aliases:"dataplanes" help:"Create dataplane (plural)."`
	Region     region.Creator    `cmd:"" help:"Create region."`
	Regions    region.Creator    `cmd:"regions" aliases:"regions" help:"Create region (plural)."`
	Workspace  workspace.Creator `cmd:"" help:"Create workspace."`
	Workspaces workspace.Creator `cmd:"workspaces" aliases:"workspaces" help:"Create workspace (plural)."`
}