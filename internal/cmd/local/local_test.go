package local

import (
	"github.com/airbytehq/abctl/internal/cmd/local/paths"
	"github.com/google/go-cmp/cmp"
	"os"
	"path/filepath"
	"testing"
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
