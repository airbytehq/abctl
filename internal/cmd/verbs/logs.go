package verbs

import (
	"github.com/airbytehq/abctl/internal/nouns/airbyte"
	"github.com/airbytehq/abctl/internal/nouns/dataplane"
	"github.com/airbytehq/abctl/internal/nouns/region"
	"github.com/airbytehq/abctl/internal/nouns/workspace"
)

type LogsCmd struct {
	Airbyte    airbyte.Logger   `cmd:"" help:"View Airbyte logs."`
	Airbytes   airbyte.Logger   `cmd:"airbytes" aliases:"airbytes" help:"View Airbyte logs (plural)."`
	Dataplane  dataplane.Logger `cmd:"" help:"View dataplane logs."`
	Dataplanes dataplane.Logger `cmd:"dataplanes" aliases:"dataplanes" help:"View dataplane logs (plural)."`
	Region     region.Logger    `cmd:"" help:"View region logs."`
	Regions    region.Logger    `cmd:"regions" aliases:"regions" help:"View region logs (plural)."`
	Workspace  workspace.Logger `cmd:"" help:"View workspace logs."`
	Workspaces workspace.Logger `cmd:"workspaces" aliases:"workspaces" help:"View workspace logs (plural)."`
}