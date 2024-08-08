package main

import (
	"context"
	"errors"
	"github.com/airbytehq/abctl/internal/build"
	"github.com/airbytehq/abctl/internal/cmd"
	"github.com/airbytehq/abctl/internal/update"
	"github.com/airbytehq/abctl/internal/ux"
	"github.com/airbytehq/abctl/internal/ux/event"
	"github.com/airbytehq/abctl/internal/ux/status"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pterm/pterm"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	m := ux.New()
	p := tea.NewProgram(m)
	go func() {
		time.Sleep(1 * time.Second)
		p.Send(status.Msg{Type: status.UPDATE, Msg: "Update message"})
		time.Sleep(1 * time.Second)
		p.Send(status.Msg{Type: status.SUCCESS, Msg: "Success message"})
		time.Sleep(1 * time.Second)
		p.Send(status.Msg{Type: status.FAILURE, Msg: "Failure message"})
		p.Send(status.Msg{Type: status.UPDATE, Msg: "Update message 0"})
		p.Send(status.Msg{Type: status.UPDATE, Msg: "Update message 1"})
		p.Send(status.Msg{Type: status.UPDATE, Msg: "Update message 2"})
		p.Send(status.Msg{Type: status.UPDATE, Msg: "Update message 3"})
		p.Send(status.Msg{Type: status.UPDATE, Msg: "Update message 4"})
		p.Send(status.Msg{Type: status.UPDATE, Msg: "Update message 5"})
		p.Send(event.Msg{Type: event.INFO, Msg: "An event occurred"})
		p.Send(status.Msg{Type: status.UPDATE, Msg: "Update message 6"})
		time.Sleep(1 * time.Second)

		p.Send(event.Msg{Type: event.INFO, Msg: "An event occurred 2"})
		p.Send(status.Msg{Type: status.UPDATE, Msg: "Update message 6"})
		p.Send(event.Msg{Type: event.WARN, Msg: "An event occurred 3"})
		p.Send(status.Msg{Type: status.UPDATE, Msg: "Update message 6"})
		p.Send(event.Msg{Type: event.INFO, Msg: "An event occurred 4"})

		p.Send(status.Msg{Type: status.UPDATE, Msg: "Update message 7"})
		p.Send(status.Msg{Type: status.UPDATE, Msg: "Update message 8"})
		p.Send(status.Msg{Type: status.UPDATE, Msg: "Update message 900"})
		p.Send(status.Msg{Type: status.SUCCESS, Msg: "Success again message"})
		time.Sleep(1 * time.Second)
		p.Send(status.Msg{Type: status.UPDATE, Msg: "Update message"})
	}()
	if _, err := p.Run(); err != nil {
		panic(err)
	}

	if true {
		return
	}

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
