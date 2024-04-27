package telemetry

var _ Client = (*NoopClient)(nil)

// NoopClient client, all methods are no-ops.
type NoopClient struct {
}

func (n NoopClient) Start(EventType) error {
	return nil
}

func (n NoopClient) Success(EventType) error {
	return nil
}

func (n NoopClient) Failure(_ EventType, _ error) error {
	return nil
}

func (n NoopClient) Attr(_, _ string) {}
