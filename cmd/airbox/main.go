package main

import (
	"context"
	"os"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/api"
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
	"github.com/pterm/pterm"
)

// rootCmd represents the airbox command.
type rootCmd struct {
	Config  config.Cmd  `cmd:"" help:"Initialize the configuration."`
	Auth    auth.Cmd    `cmd:"" help:"Authenticate with Airbyte."`
	Get     get.Cmd     `cmd:"" help:"Get Airbyte resources."`
	Delete  delete.Cmd  `cmd:"" help:"Delete Airbyte resources."`
	Install install.Cmd `cmd:"" help:"Install an Airbyte dataplane."`
}

func (c *rootCmd) BeforeApply(ctx context.Context, kCtx *kong.Context) error {
	kCtx.BindTo(k8s.DefaultProvider, (*k8s.Provider)(nil))
	kCtx.BindTo(helm.DefaultFactory, (*helm.Factory)(nil))
	kCtx.BindTo(http.DefaultClient, (*http.HTTPDoer)(nil))
	kCtx.BindTo(api.NewFactory, (*api.Factory)(nil))
	kCtx.BindTo(airbox.DefaultConfigProvider, (*airbox.ConfigProvider)(nil))
	kCtx.BindTo(ui.New(), (*ui.Provider)(nil))
	kCtx.BindTo(airboxauth.DefaultStateGenerator, (*airboxauth.StateGenerator)(nil))
	return nil
}

func main() {
	pterm.Info.Prefix.Text = " INFO  "
	ctx := context.Background()
	uiProvider := ui.New()
	var cmd rootCmd

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
