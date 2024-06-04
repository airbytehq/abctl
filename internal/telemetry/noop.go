package telemetry

import (
	"context"
	"github.com/google/uuid"
)

var _ Client = (*NoopClient)(nil)

// NoopClient client, all methods are no-ops.
type NoopClient struct {
}

func (n NoopClient) Start(context.Context, EventType) error {
	return nil
}

func (n NoopClient) Success(context.Context, EventType) error {
	return nil
}

func (n NoopClient) Failure(context.Context, EventType, error) error {
	return nil
}

func (n NoopClient) Attr(_, _ string) {}

func (n NoopClient) User() uuid.UUID {
	return uuid.Nil
}
