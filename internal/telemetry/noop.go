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

func (n NoopClient) Failure(eventType EventType, err error) error {
	return nil
}

func (n NoopClient) Attr(key, val string) {}
