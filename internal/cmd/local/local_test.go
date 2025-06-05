package local

import (
	"context"
	"errors"

	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/local"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/alecthomas/kong"

	"github.com/airbytehq/abctl/internal/cmd/local/paths"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/google/go-cmp/cmp"
)

func TestCheckAirbyteDir(t *testing.T) {
	origDir := paths.Airbyte
	t.Cleanup(func() {
		paths.Airbyte = origDir
	})

	t.Run("no directory", func(t *testing.T) {
		paths.Airbyte = filepath.Join(t.TempDir(), "does-not-exist")
		if err := checkAirbyteDir(); err != nil {
			t.Error("unexpected error", err)
		}
	})

	t.Run("directory with correct permissions", func(t *testing.T) {
		paths.Airbyte = filepath.Join(t.TempDir(), "correct-perms")
		if err := os.MkdirAll(paths.Airbyte, 0744); err != nil {
			t.Fatal("unable to create test directory", err)
		}
		if err := os.Chmod(paths.Airbyte, 0744); err != nil {
			t.Fatal("unable to change permissions", err)
		}
		if err := checkAirbyteDir(); err != nil {
			t.Error("unexpected error", err)
		}

		// permissions should be unchanged
		perms, err := os.Stat(paths.Airbyte)
		if err != nil {
			t.Fatal("unable to check permissions", err)
		}
		if d := cmp.Diff(0744, int(perms.Mode().Perm())); d != "" {
			t.Errorf("permissions mismatch (-want +got):\n%s", d)
		}
	})

	t.Run("directory with higher permissions", func(t *testing.T) {
		paths.Airbyte = filepath.Join(t.TempDir(), "correct-perms")
		if err := os.MkdirAll(paths.Airbyte, 0777); err != nil {
			t.Fatal("unable to create test directory", err)
		}
		if err := os.Chmod(paths.Airbyte, 0777); err != nil {
			t.Fatal("unable to change permissions", err)
		}
		if err := checkAirbyteDir(); err != nil {
			t.Error("unexpected error", err)
		}

		// permissions should be unchanged
		perms, err := os.Stat(paths.Airbyte)
		if err != nil {
			t.Fatal("unable to check permissions", err)
		}
		if d := cmp.Diff(0777, int(perms.Mode().Perm())); d != "" {
			t.Errorf("permissions mismatch (-want +got):\n%s", d)
		}
	})

	t.Run("directory with incorrect permissions", func(t *testing.T) {
		paths.Airbyte = filepath.Join(t.TempDir(), "incorrect-perms")
		if err := os.MkdirAll(paths.Airbyte, 0200); err != nil {
			t.Fatal("unable to create test directory", err)
		}
		if err := os.Chmod(paths.Airbyte, 0200); err != nil {
			t.Fatal("unable to change permissions", err)
		}
		// although the permissions are incorrect, checkAirbyteDir should fix them
		if err := checkAirbyteDir(); err != nil {
			t.Fatal("unexpected error", err)
		}

		// permissions should be changed
		perms, err := os.Stat(paths.Airbyte)
		if err != nil {
			t.Fatal("unable to check permissions", err)
		}
		if d := cmp.Diff(0744, int(perms.Mode().Perm())); d != "" {
			t.Errorf("permissions mismatch (-want +got):\n%s", d)
		}
	})
}

func TestValues_FileDoesntExist(t *testing.T) {

	var root InstallCmd
	k, _ := kong.New(
		&root,
		kong.Name("abctl"),
		kong.Description("Airbyte's command line tool for managing a local Airbyte installation."),
		kong.UsageOnError(),
	)
	_, err := k.Parse([]string{"--values", "/testdata/thisfiledoesnotexist"})
	if err == nil {
		t.Fatal("expected error")
	}
	expect := "--values: stat /testdata/thisfiledoesnotexist: no such file or directory"
	if err.Error() != expect {
		t.Errorf("expected %q but got %q", expect, err)
	}
}

func TestValues_BadYaml(t *testing.T) {

	cmd := InstallCmd{Values: "./local/testdata/invalid.values.yaml"}
	err := cmd.Run(context.Background(), k8s.TestProvider, telemetry.NoopClient{})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.HasPrefix(err.Error(), "failed to unmarshal file") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInvalidHostFlag_IpAddr(t *testing.T) {
	cmd := InstallCmd{Host: []string{"ok", "1.2.3.4"}}
	err := cmd.Run(context.Background(), k8s.TestProvider, telemetry.NoopClient{})
	if !errors.Is(err, localerr.ErrIpAddressForHostFlag) {
		t.Errorf("expected ErrIpAddressForHostFlag but got %v", err)
	}
}

func TestInvalidHostFlag_IpAddrWithPort(t *testing.T) {
	cmd := InstallCmd{Host: []string{"ok", "1.2.3.4:8000"}}
	err := cmd.Run(context.Background(), k8s.TestProvider, telemetry.NoopClient{})
	if !errors.Is(err, localerr.ErrInvalidHostFlag) {
		t.Errorf("expected ErrInvalidHostFlag but got %v", err)
	}
}

func TestInstallOpts(t *testing.T) {
	b, _ := os.ReadFile("local/testdata/expected-default.values.yaml")
	cmd := InstallCmd{
		// Don't let the code dynamically resolve the latest chart version.
		Chart: "/test/path/to/chart",
	}
	expect := &local.InstallOpts{
		HelmValuesYaml:  string(b),
		AirbyteChartLoc: "/test/path/to/chart",
		LocalStorage:    true,
	}
	opts, err := cmd.InstallOpts(context.Background(), "test-user")
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(expect, opts); d != "" {
		t.Errorf("unexpected error diff (-want +got):\n%s", d)
	}
}
