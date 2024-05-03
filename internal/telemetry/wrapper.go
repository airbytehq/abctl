package telemetry

import (
	"context"
	"github.com/pterm/pterm"
)

// Wrapper wraps the function calls with the telemetry handlers
func Wrapper(ctx context.Context, et EventType, f func() error) (err error) {
	cli := Get()

	attemptSuccessFailure := true

	if err := cli.Start(ctx, et); err != nil {
		pterm.Debug.Printfln("Unable to send telemetry start data: %s", err)
		attemptSuccessFailure = false
	}

	defer func() {
		if !attemptSuccessFailure {
			return
		}

		if err != nil {
			if err := cli.Failure(ctx, et, err); err != nil {
				pterm.Debug.Printfln("Unable to send telemetry failure data: %s", err)
			}
		} else {
			if err := cli.Success(ctx, et); err != nil {
				pterm.Debug.Printfln("Unable to send telemetry success data: %s", err)
			}
		}
	}()

	return f()
}
