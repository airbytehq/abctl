package telemetry

var _ Client = (*MockClient)(nil)

type MockClient struct {
	attrs   map[string]string
	start   func(EventType) error
	success func(EventType) error
	failure func(EventType, error) error
	wrap    func(EventType, func() error) error
}

func (m *MockClient) Start(eventType EventType) error {
	return m.start(eventType)
}

func (m *MockClient) Success(eventType EventType) error {
	return m.success(eventType)
}

func (m *MockClient) Failure(eventType EventType, err error) error {
	return m.failure(eventType, err)
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

func (m *MockClient) Wrap(et EventType, f func() error) error {
	return m.wrap(et, f)
}
