package k8s

import (
	"bytes"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pterm/pterm"
)

func TestLogger_HandleWarningHeader(t *testing.T) {
	b := bytes.NewBufferString("")
	pterm.SetDefaultOutput(b)
	pterm.EnableDebugMessages()
	// remove color codes from output
	pterm.DisableColor()
	t.Cleanup(func() {
		pterm.SetDefaultOutput(os.Stdout)
		pterm.DisableDebugMessages()
		pterm.EnableColor()
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
			want: "  DEBUG   k8s - WARN: test msg\n",
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
