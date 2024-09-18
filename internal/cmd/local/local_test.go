package local

import (
	"context"
	"errors"

	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"

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
	cmd := InstallCmd{Values: "thisfiledoesnotexist"}
	err := cmd.Run(context.Background(), k8s.TestProvider, telemetry.NoopClient{})
	if err == nil {
		t.Fatal("expected error")
	}
	expect := "failed to read file thisfiledoesnotexist: open thisfiledoesnotexist: no such file or directory"
	if err.Error() != expect {
		t.Errorf("expected %q but got %q", expect, err)
	}
}

func TestValues_BadYaml(t *testing.T) {
	tmpdir := t.TempDir()
	p := filepath.Join(tmpdir, "values.yaml")
	content := `
foo:
  - bar: baz
    - foo
`

	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := InstallCmd{Values: p}
	err := cmd.Run(context.Background(), k8s.TestProvider, telemetry.NoopClient{})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.HasPrefix(err.Error(), "failed to unmarshal file") {
		t.Errorf("unexpected error: %v", err)

	}
}

func TestInvalidHostFlag(t *testing.T) {
	cmd := InstallCmd{Host: []string{"ok", "1.2.3.4"}}
	err := cmd.Run(context.Background(), k8s.TestProvider, telemetry.NoopClient{})
	if !errors.Is(err, localerr.ErrIpAddressForHostFlag) {
		t.Errorf("expected ErrIpAddressForHostFlag but got %v", err)
	}
}
