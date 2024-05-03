package telemetry

import (
	"context"
	"github.com/pterm/pterm"
)

// Wrapper wraps the function calls with the telemetry handlers
func Wrapper(ctx context.Context, et EventType, f func() error) error {
	cli := Get()

	attemptSuccessFailure := true

	if err := cli.Start(ctx, et); err != nil {
		pterm.Debug.Printfln("Unable to send telemetry start data: %s", err)
		attemptSuccessFailure = false
	}

	if err := f(); err != nil {
		if attemptSuccessFailure {
			if errTel := cli.Failure(ctx, et, err); errTel != nil {
				pterm.Debug.Printfln("Unable to send telemetry failure data: %s", errTel)
			}
		}

		return err
	}

	if attemptSuccessFailure {
		if err := cli.Success(ctx, et); err != nil {
			pterm.Debug.Printfln("Unable to send telemetry success data: %s", err)
		}
	}

	return nil
}
