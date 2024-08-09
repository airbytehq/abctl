package k8s

import (
	"fmt"
	"github.com/airbytehq/abctl/internal/status"
	"k8s.io/client-go/rest"
)

var _ rest.WarningHandler = (*Logger)(nil)

// Logger is an implementation of the WarningHandler that converts the k8s warning messages
// into abctl debug messages.
type Logger struct {
}

func (x Logger) HandleWarningHeader(code int, _ string, msg string) {
	// code and length check are taken from the default WarningLogger implementation
	if code != 299 || len(msg) == 0 {
		return
	}
	status.Debug(fmt.Sprintf("k8s - WARN: %s", msg))
}
