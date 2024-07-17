package telemetry

import (
	"context"
	"errors"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"testing"
)

func TestNoopClient(t *testing.T) {
	cli := NoopClient{}
	ctx := context.Background()
	if err := cli.Start(ctx, Install); err != nil {
		t.Error(err)
	}
	if err := cli.Success(ctx, Install); err != nil {
		t.Error(err)
	}
	if err := cli.Failure(ctx, Install, errors.New("")); err != nil {
		t.Error(err)
	}

	cli.Attr("k", "v'")
}

// Verify that the func() error is actually called for the NoopClient.Wrap
func TestNoopClient_Wrap(t *testing.T) {
	t.Run("fn is called without error", func(t *testing.T) {
		called := false
		fn := func() error {
			called = true
			return nil
		}

		cli := NoopClient{}

		if err := cli.Wrap(context.Background(), Install, fn); err != nil {
			t.Fatal("unexpected error", err)
		}

		if d := cmp.Diff(true, called); d != "" {
			t.Errorf("function should have been called (-want, +got): %s", d)
		}
	})

	t.Run("fn is called with error", func(t *testing.T) {
		called := false
		expectedErr := errors.New("test")
		fn := func() error {
			called = true
			return expectedErr
		}

		cli := NoopClient{}

		err := cli.Wrap(context.Background(), Install, fn)
		if d := cmp.Diff(expectedErr, err, cmpopts.EquateErrors()); d != "" {
			t.Errorf("function should have returned an error (-want, +got): %s", d)
		}

		if d := cmp.Diff(true, called); d != "" {
			t.Errorf("function should have been called (-want, +got): %s", d)
		}
	})
}
