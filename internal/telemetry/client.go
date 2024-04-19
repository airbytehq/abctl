package telemetry

type EventState string

const (
	Start   EventState = "started"
	Failed  EventState = "failed"
	Success EventState = "succeeded"
)

type EventType string

const (
	Install   EventType = "install"
	Uninstall EventType = "uninstall"
)

// Client interface for telemetry data.
type Client interface {
	// Start should be called as soon quickly as possible.
	Start(EventType) error
	// Success should be called only if the activity succeeded.
	Success(EventType) error
	// Failure should be called only if the activity failed.
	Failure(EventType, error) error
	// Attr should be called to add additional attributes to this activity.
	Attr(key, val string)
}
