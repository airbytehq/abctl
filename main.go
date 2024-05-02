package main

import (
	"context"
	"github.com/airbytehq/abctl/internal/cmd"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
		<-signalCh

		cancel()
	}()

	root := cmd.NewCmd()
	cmd.Execute(ctx, root)
}
