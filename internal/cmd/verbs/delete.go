package verbs

import (
	"github.com/airbytehq/abctl/internal/nouns/airbyte"
	"github.com/airbytehq/abctl/internal/nouns/dataplane"
	"github.com/airbytehq/abctl/internal/nouns/region"
	"github.com/airbytehq/abctl/internal/nouns/workspace"
)

type DeleteCmd struct {
	Airbyte    airbyte.Deleter   `cmd:"" help:"Delete Airbyte installation."`
	Airbytes   airbyte.Deleter   `cmd:"airbytes" aliases:"airbytes" help:"Delete Airbyte installation (plural)."`
	Dataplane  dataplane.Deleter `cmd:"" help:"Delete dataplane."`
	Dataplanes dataplane.Deleter `cmd:"dataplanes" aliases:"dataplanes" help:"Delete dataplane (plural)."`
	Region     region.Deleter    `cmd:"" help:"Delete region."`
	Regions    region.Deleter    `cmd:"regions" aliases:"regions" help:"Delete region (plural)."`
	Workspace  workspace.Deleter `cmd:"" help:"Delete workspace."`
	Workspaces workspace.Deleter `cmd:"workspaces" aliases:"workspaces" help:"Delete workspace (plural)."`
}