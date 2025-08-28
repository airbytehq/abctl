package main

import (
	"context"
	"os"

	"github.com/airbytehq/abctl/internal/cmd/auth"
	"github.com/airbytehq/abctl/internal/cmd/config"
	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/alecthomas/kong"
	"github.com/pterm/pterm"
)

// rootCmd represents the airbox command.
type rootCmd struct {
	Config config.Cmd `cmd:"" help:"Initialize the configuration."`
	Auth   auth.Cmd   `cmd:"" help:"Authenticate with Airbyte."`
}

func main() {
	pterm.Info.Prefix.Text = " INFO  "
	ctx := context.Background()
	var cmd rootCmd

	parser, err := kong.New(
		&cmd,
		kong.Name("airbox"),
		kong.Bind(k8s.DefaultProvider, (*k8s.Provider)(nil)),
	)
	if err != nil {
		panic(err)
	}

	parsed, err := parser.Parse(os.Args[1:])
	if err != nil {
		parser.FatalIfErrorf(err)
	}

	parsed.BindToProvider(func() (context.Context, error) {
		return ctx, nil
	})

	err = parsed.Run()
	if err != nil {
		pterm.Error.Println(err)
	}
}
