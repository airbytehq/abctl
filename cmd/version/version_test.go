package version

import (
	"bytes"
	"fmt"
	"github.com/airbytehq/abctl/internal/build"
	"github.com/google/go-cmp/cmp"
	"github.com/pterm/pterm"
	"os"
	"testing"
)

func TestCmd_Output(t *testing.T) {
	b := bytes.NewBufferString("")
	pterm.SetDefaultOutput(b)
	t.Cleanup(func() {
		pterm.SetDefaultOutput(os.Stdout)
	})

	if err := Cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	exp := fmt.Sprintf("version: %s\nrevision: %s\ntime: %s\nmodified: %t\n", build.Version, build.Revision, build.ModificationTime, build.Modified)

	if d := cmp.Diff(exp, b.String()); d != "" {
		t.Error("cmd mismatch (-want +got):\n", d)
	}
}

func TestCmd_OutputOverride(t *testing.T) {
	origVersion := build.Version
	build.Version = "v12.15.82"
	build.Revision = "revision"
	build.Modified = true
	build.ModificationTime = "20240401T00:00:00Z"
	b := bytes.NewBufferString("")
	pterm.SetDefaultOutput(b)

	t.Cleanup(func() {
		pterm.SetDefaultOutput(os.Stdout)
		build.Version = origVersion
		build.Revision = ""
		build.Modified = false
		build.ModificationTime = ""
	})

	exp := fmt.Sprintf("version: %s\nrevision: %s\ntime: %s\nmodified: %t\n", build.Version, build.Revision, build.ModificationTime, build.Modified)

	if err := Cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if d := cmp.Diff(exp, b.String()); d != "" {
		t.Error("cmd mismatch (-want +got):\n", d)
	}
}
