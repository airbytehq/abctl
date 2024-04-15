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

	if d := cmp.Diff(fmt.Sprintf("version: %s\n", build.Version), b.String()); d != "" {
		t.Error("cmd mismatch (-want +got):\n", d)
	}
}

func TestCmd_OutputVersionOverride(t *testing.T) {
	origVersion := build.Version
	build.Version = "v12.15.82"
	b := bytes.NewBufferString("")
	pterm.SetDefaultOutput(b)

	t.Cleanup(func() {
		pterm.SetDefaultOutput(os.Stdout)
		build.Version = origVersion
	})

	if err := Cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if d := cmp.Diff(fmt.Sprintf("version: %s\n", build.Version), b.String()); d != "" {
		t.Error("cmd mismatch (-want +got):\n", d)
	}
}
