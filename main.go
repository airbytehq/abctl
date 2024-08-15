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

	root := cmd.NewCmd()
	cmd.Execute(ctx, root)

	newRelease := <-updateChan
	if newRelease.err != nil {
		if errors.Is(newRelease.err, update.ErrDevVersion) {
			pterm.DefaultLogger.Debug("Release checking is disabled for dev builds")
		}
	} else if newRelease.version != "" {
		pterm.Println()
		pterm.Info.Printfln("A new release of abctl is available: %s -> %s\nUpdating to the latest version is highly recommended", build.Version, newRelease.version)
	}
}

type updateInfo struct {
	version string
	err     error
}
