package telemetry

import (
	"context"
	"errors"
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
