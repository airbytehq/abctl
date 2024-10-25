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

func (n NoopClient) Failure(EventType, error) error {
	return nil
}

func (n NoopClient) Attr(_, _ string) {}

func (n NoopClient) User() string {
	return ""
}

func (n NoopClient) Wrap(et EventType, f func() error) error {
	return f()
}
