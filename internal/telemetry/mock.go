package telemetry

import "context"

var _ Client = (*MockClient)(nil)

type MockClient struct {
	attrs   map[string]string
	start   func(context.Context, EventType) error
	success func(context.Context, EventType) error
	failure func(context.Context, EventType, error) error
	wrap    func(context.Context, EventType, func() error) error
}

func (m *MockClient) Start(ctx context.Context, eventType EventType) error {
	return m.start(ctx, eventType)
}

func (m *MockClient) Success(ctx context.Context, eventType EventType) error {
	return m.success(ctx, eventType)
}

func (m *MockClient) Failure(ctx context.Context, eventType EventType, err error) error {
	return m.failure(ctx, eventType, err)
}

func (m *MockClient) Attr(key, val string) {
	if m.attrs == nil {
		m.attrs = map[string]string{}
	}
	m.attrs[key] = val
}

func (m *MockClient) User() string {
	return "test-user"
}

func (m *MockClient) Wrap(ctx context.Context, et EventType, f func() error) error {
	return m.wrap(ctx, et, f)
}
