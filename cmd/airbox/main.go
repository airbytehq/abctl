package main

import (
	"context"
	"os"

	"github.com/airbytehq/abctl/internal/airbox"
	airboxauth "github.com/airbytehq/abctl/internal/auth"
	"github.com/airbytehq/abctl/internal/cmd/auth"
	"github.com/airbytehq/abctl/internal/cmd/config"
	"github.com/airbytehq/abctl/internal/cmd/delete"
	"github.com/airbytehq/abctl/internal/cmd/get"
	"github.com/airbytehq/abctl/internal/cmd/install"
	"github.com/airbytehq/abctl/internal/helm"
	"github.com/airbytehq/abctl/internal/http"
	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/ui"
	"github.com/alecthomas/kong"
)

// rootCmd represents the airbox command.
type rootCmd struct {
	Config  config.Cmd  `cmd:"" help:"Initialize the configuration."`
	Auth    auth.Cmd    `cmd:"" help:"Authenticate with Airbyte."`
	Get     get.Cmd     `cmd:"" help:"Get Airbyte resources."`
	Delete  delete.Cmd  `cmd:"" help:"Delete Airbyte resources."`
	Install install.Cmd `cmd:"" help:"Install an Airbyte dataplane."`
}

// Global UI provider for terminal output
var uiProvider ui.Provider

func init() {
	uiProvider = ui.New()
}

func (c *rootCmd) BeforeApply(ctx context.Context, kCtx *kong.Context) error {
	kCtx.BindTo(&airbox.FileConfigStore{}, (*airbox.ConfigStore)(nil))
	kCtx.BindTo(http.DefaultClient, (*http.HTTPDoer)(nil))
	kCtx.BindTo(airbox.NewAPIService, (*airbox.APIServiceFactory)(nil))
	kCtx.BindTo(helm.DefaultFactory, (*helm.Factory)(nil))
	kCtx.BindTo(k8s.DefaultClusterFactory, (*k8s.ClusterFactory)(nil))
	kCtx.BindTo(uiProvider, (*ui.Provider)(nil))
	kCtx.BindTo(airboxauth.DefaultStateGenerator, (*airboxauth.StateGenerator)(nil))
	return nil
}

func main() {
	ctx := context.Background()
	cmd := rootCmd{}

	parser, err := kong.New(
		&cmd,
		kong.Name("airbox"),
		kong.BindToProvider(bindCtx(ctx)),
	)
	if err != nil {
		panic(err)
	}

	parsed, err := parser.Parse(os.Args[1:])
	if err != nil {
		parser.FatalIfErrorf(err)
	}

	err = parsed.BindToProvider(func() (context.Context, error) {
		return ctx, nil
	})
	if err != nil {
		panic(err)
	}

	err = parsed.Run()
	if err != nil {
		uiProvider.NewLine()
		uiProvider.ShowError(err)
		uiProvider.NewLine()
	}
}

// bindCtx exists to allow kong to correctly inject a context.Context into the Run methods on the commands.
func bindCtx(ctx context.Context) func() (context.Context, error) {
	return func() (context.Context, error) {
		return ctx, nil
	}
}
