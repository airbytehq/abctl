package k8s

import (
	"bytes"
	"github.com/airbytehq/abctl/internal/status"
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestLogger_HandleWarningHeader(t *testing.T) {
	b := bytes.NewBufferString("")
	origWriter := status.SetWriter(b)
	origDebug := status.SetDebug(true)
	t.Cleanup(func() {
		status.SetWriter(origWriter)
		status.SetDebug(origDebug)
	})

	tests := []struct {
		name string
		code int
		msg  string
		want string
	}{
		{
			name: "non 299 code",
			code: 300,
			msg:  "test msg",
		},
		{
			name: "empty msg",
			code: 299,
		},
		{
			name: "happy path",
			code: 299,
			msg:  "test msg",
			want: "DEBU k8s - WARN: test msg\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := Logger{}
			logger.HandleWarningHeader(tt.code, "", tt.msg)

			if d := cmp.Diff(tt.want, b.String()); d != "" {
				t.Error("unexpected output (-want, +got) =", d)
			}
		})
	}
}
