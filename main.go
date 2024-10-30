package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/airbytehq/abctl/internal/build"
	"github.com/airbytehq/abctl/internal/cmd"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/airbytehq/abctl/internal/trace"
	"github.com/airbytehq/abctl/internal/update"
	"github.com/alecthomas/kong"
	"github.com/pterm/pterm"
)

func main() {
	os.Exit(run())
}

// run is essentially the main method returning the exitCode of the program.
// Run is separated to ensure that deferred functions are called (os.Exit prevents this).
func run() int {
	// ensure the pterm info width matches the other printers
	pterm.Info.Prefix.Text = " INFO  "

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	printUpdateMsg := checkForNewerAbctlVersion(ctx)

	telClient := telemetry.Get()

	shutdowns, err := trace.Init(ctx, telClient.User())
	if err != nil {
		pterm.Debug.Printf(fmt.Sprintf("Trace disabled: %s", err))
	}
	defer func() {
		for _, shutdown := range shutdowns {
			shutdown()
		}
	}()

	runCmd := func(ctx context.Context) error {
		var root cmd.Cmd
		parser, err := kong.New(
			&root,
			kong.Name("abctl"),
			kong.Description("Airbyte's command line tool for managing a local Airbyte installation."),
			kong.UsageOnError(),
			kong.BindToProvider(bindCtx(ctx)),
			kong.BindTo(telClient, (*telemetry.Client)(nil)),
		)
		if err != nil {
			return err
		}
		parsed, err := parser.Parse(os.Args[1:])
		if err != nil {
			return err
		}

		ctx, span := trace.NewSpan(ctx, fmt.Sprintf("abctl %s", parsed.Command()))
		defer span.End()

		parsed.BindToProvider(bindCtx(ctx))
		return parsed.Run()
	}

	exitCode := handleErr(ctx, runCmd(ctx))
	printUpdateMsg()
	return exitCode
}

func handleErr(ctx context.Context, err error) int {
	if err == nil {
		return 0
	}

	trace.CaptureError(ctx, err)

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

// bindCtx exists to allow kong to correctly inject a context.Context into the Run methods on the commands.
func bindCtx(ctx context.Context) func() (context.Context, error) {
	return func() (context.Context, error) {
		return ctx, nil
	}
}
