package pgdata

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestVersion_ReadsVersion(t *testing.T) {
	dir := t.TempDir()
	pgDataDir := filepath.Join(dir, "pgdata")
	if err := os.MkdirAll(pgDataDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pgDataDir, "PG_VERSION"), []byte("17\n"), 0644); err != nil {
		t.Fatal(err)
	}

	c := New(&Config{Path: pgDataDir})
	ver, err := c.Version()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "17" {
		t.Fatalf("expected version 17, got %q", ver)
	}
}

func TestVersion_NonExistentDir(t *testing.T) {
	c := New(&Config{Path: filepath.Join(t.TempDir(), "does-not-exist")})
	_, err := c.Version()
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected fs.ErrNotExist in error chain, got: %v", err)
	}
}

func TestVersion_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	pgDataDir := filepath.Join(dir, "pgdata")
	if err := os.MkdirAll(pgDataDir, 0755); err != nil {
		t.Fatal(err)
	}
	versionFile := filepath.Join(pgDataDir, "PG_VERSION")
	if err := os.WriteFile(versionFile, []byte("16\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Remove all permissions from the parent directory so the file cannot be read.
	if err := os.Chmod(pgDataDir, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// Restore permissions so t.TempDir() cleanup can remove the directory.
		os.Chmod(pgDataDir, 0755)
	})

	c := New(&Config{Path: pgDataDir})
	_, err := c.Version()
	if err == nil {
		t.Fatal("expected error for permission denied, got nil")
	}
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("expected fs.ErrPermission in error chain, got: %v", err)
	}
}

func TestVersion_OlderVersion(t *testing.T) {
	dir := t.TempDir()
	pgDataDir := filepath.Join(dir, "pgdata")
	if err := os.MkdirAll(pgDataDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pgDataDir, "PG_VERSION"), []byte("13\n"), 0644); err != nil {
		t.Fatal(err)
	}

	c := New(&Config{Path: pgDataDir})
	ver, err := c.Version()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "13" {
		t.Fatalf("expected version 13, got %q", ver)
	}
}
