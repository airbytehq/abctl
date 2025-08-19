package verbs

import (
	"github.com/airbytehq/abctl/internal/nouns/airbyte"
	"github.com/airbytehq/abctl/internal/nouns/dataplane"
	"github.com/airbytehq/abctl/internal/nouns/region"
	"github.com/airbytehq/abctl/internal/nouns/workspace"
)

type GetCmd struct {
	Airbyte   airbyte.Getter   `cmd:"" help:"Get Airbyte information."`
	Airbytes  airbyte.Getter   `cmd:"airbytes" aliases:"airbytes" help:"Get Airbyte information (plural)."`
	Dataplane  dataplane.Getter `cmd:"" help:"Get dataplane information."`
	Dataplanes dataplane.Getter `cmd:"dataplanes" aliases:"dataplanes" help:"Get dataplane information (plural)."`
	Region     region.Getter    `cmd:"" help:"Get region information."`
	Regions    region.Getter    `cmd:"regions" aliases:"regions" help:"Get region information (plural)."`
	Workspace  workspace.Getter `cmd:"" help:"Get workspace information."`
	Workspaces workspace.Getter `cmd:"workspaces" aliases:"workspaces" help:"Get workspace information (plural)."`
}