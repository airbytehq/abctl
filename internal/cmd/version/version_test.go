package version

import (
	"bytes"
	"os"
	"testing"

	"github.com/airbytehq/abctl/internal/build"
	"github.com/google/go-cmp/cmp"
	"github.com/pterm/pterm"
)

func TestCmd_Output(t *testing.T) {
	b := bytes.NewBufferString("")
	pterm.SetDefaultOutput(b)
	t.Cleanup(func() {
		pterm.SetDefaultOutput(os.Stdout)
	})

	tests := []struct {
		name             string
		version          string
		revision         string
		modificationTime string
		modified         bool
		expected         string
	}{
		{
			name:     "version defined",
			version:  "v0.0.0",
			expected: "version: v0.0.0\n",
		},
		{
			name:     "revision defined",
			version:  "v0.0.0",
			revision: "d34db33f",
			expected: "version: v0.0.0\nrevision: d34db33f\n",
		},
		{
			name:             "modification time defined",
			version:          "v0.0.0",
			modificationTime: "time-goes-here",
			expected:         "version: v0.0.0\ntime: time-goes-here\n",
		},
		{
			name:     "modified defined",
			version:  "v0.0.0",
			modified: true,
			expected: "version: v0.0.0\nmodified: true\n",
		},
		{
			name:             "everything defined",
			version:          "v0.0.0",
			revision:         "d34db33f",
			modificationTime: "time-goes-here",
			modified:         true,
			expected:         "version: v0.0.0\nrevision: d34db33f\ntime: time-goes-here\nmodified: true\n",
		},
	}

	origVersion := build.Version
	origRevision := build.Revision
	origModification := build.ModificationTime
	origModified := build.Modified

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			t.Cleanup(func() {
				build.Version = origVersion
				build.Revision = origRevision
				build.ModificationTime = origModification
				build.Modified = origModified
				b.Reset()
			})

			build.Version = tt.version
			build.Revision = tt.revision
			build.ModificationTime = tt.modificationTime
			build.Modified = tt.modified

			cmd := NewCmdVersion()

			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}

			if d := cmp.Diff(tt.expected, b.String()); d != "" {
				t.Error("cmd mismatch (-want +got):\n", d)
			}
		})
	}
}
