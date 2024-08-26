package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/airbytehq/abctl/internal/build"
	"github.com/airbytehq/abctl/internal/cmd"
	"github.com/airbytehq/abctl/internal/update"
	"github.com/alecthomas/kong"
	"github.com/pterm/pterm"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// check for update
	updateCtx, updateCancel := context.WithTimeout(ctx, 2*time.Second)
	defer updateCancel()

	updateChan := make(chan updateInfo)
	go func() {
		info := updateInfo{}
		info.version, info.err = update.Check(updateCtx, http.DefaultClient, build.Version)
		updateChan <- info
	}()

	// listen for shutdown signals
	go func() {
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
		<-signalCh

		cancel()
	}()

	// ensure the pterm info width matches the other printers
	pterm.Info.Prefix.Text = " INFO  "

	var root cmd.Cmd
	parser, err := kong.New(
		&root,
		kong.Name("abctl"),
		kong.Description("Airbyte's command line tool for managing a local Airbyte installation."),
		kong.UsageOnError(),
	)
	if err != nil {
		cmd.HandleErr(err)
	}
	parsed, err := parser.Parse(os.Args[1:])
	if err != nil {
		cmd.HandleErr(err)
	}
	if err := parsed.BindToProvider(bindCtx(ctx)); err != nil {
		cmd.HandleErr(err)
	}

	cmd.HandleErr(parsed.Run())

	newRelease := <-updateChan
	if newRelease.err != nil {
		if errors.Is(newRelease.err, update.ErrDevVersion) {
			pterm.Debug.Println("Release checking is disabled for dev builds")
		}
	} else if newRelease.version != "" {
		pterm.Println()
		pterm.Info.Printfln("A new release of abctl is available: %s -> %s\nUpdating to the latest version is highly recommended", build.Version, newRelease.version)
	}
}

func bindCtx(ctx context.Context) func() (context.Context, error) {
	return func() (context.Context, error) {
		return ctx, nil
	}
}

type updateInfo struct {
	version string
	err     error
}
