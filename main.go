package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/airbytehq/abctl/internal/build"
	"github.com/airbytehq/abctl/internal/cmd"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/update"
	"github.com/alecthomas/kong"
	"github.com/pterm/pterm"
)

func main() {
	// ensure the pterm info width matches the other printers
	pterm.Info.Prefix.Text = " INFO  "

	ctx := cliCtx()
	printUpdateMsg := checkForNewerAbctlVersion(ctx)
	exitCode := handleErr(run(ctx))
	printUpdateMsg()

	os.Exit(exitCode)
}

func run(ctx context.Context) error {
	var root cmd.Cmd
	parser, err := kong.New(
		&root,
		kong.Name("abctl"),
		kong.Description("Airbyte's command line tool for managing a local Airbyte installation."),
		kong.UsageOnError(),
	)
	if err != nil {
		return err
	}
	parsed, err := parser.Parse(os.Args[1:])
	if err != nil {
		return err
	}
	parsed.BindToProvider(bindCtx(ctx))
	return parsed.Run()
}

func handleErr(err error) int {
	if err == nil {
		return 0
	}

	pterm.Error.Println(err)

	var errParse *kong.ParseError
	if errors.As(err, &errParse) {
		_ = kong.DefaultHelpPrinter(kong.HelpOptions{}, errParse.Context)
	}

	var e *localerr.LocalError
	if errors.As(err, &e) {
		pterm.Println()
		pterm.Info.Println(e.Help())
	}

	return 1
}

// checkForNewerAbctlVersion checks for a newer version of abctl.
// Returns a function that, when called, will display a message if a newer version is available.
func checkForNewerAbctlVersion(ctx context.Context) func() {
	c := make(chan string)
	go func() {
		defer close(c)
		ver, err := update.Check(ctx)
		if err != nil {
			pterm.Debug.Printfln("update check: %s", err)
		} else {
			c <- ver
		}
	}()

	return func() {
		ver := <-c
		if ver != "" {
			pterm.Info.Printfln("A new release of abctl is available: %s -> %s\nUpdating to the latest version is highly recommended", build.Version, ver)

		}
	}
}

// cliCtx configures and returns a context which listens for interrupt/shutdown signals.
func cliCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	// listen for shutdown signals
	go func() {
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
		<-signalCh

		cancel()
	}()
	return ctx
}

// bindCtx exists to allow kong to correctly inject a context.Context into the Run methods on the commands.
func bindCtx(ctx context.Context) func() (context.Context, error) {
	return func() (context.Context, error) {
		return ctx, nil
	}
}
