package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/airbytehq/abctl/internal/paths"
)

func TestDefaultK8s_KubeconfigHandling(t *testing.T) {
	// Save original KUBECONFIG env var
	originalKubeconfig := os.Getenv("KUBECONFIG")
	defer func() {
		os.Setenv("KUBECONFIG", originalKubeconfig)
	}()

	tests := []struct {
		name        string
		kubecfg     string
		kubectx     string
		setupEnv    func()
		wantErr     bool
		errContains string
	}{
		{
			name:    "explicit kubeconfig path",
			kubecfg: "/tmp/test-kubeconfig",
			kubectx: "",
			setupEnv: func() {
				os.Unsetenv("KUBECONFIG")
			},
			wantErr:     true,
			errContains: "no such file or directory",
		},
		{
			name:    "empty kubeconfig uses default resolution",
			kubecfg: "",
			kubectx: "",
			setupEnv: func() {
				// Test with KUBECONFIG env var set
				tmpDir := t.TempDir()
				testConfig := filepath.Join(tmpDir, "test.config")
				os.Setenv("KUBECONFIG", testConfig)
			},
			wantErr:     true,
			errContains: "invalid configuration",
		},
		{
			name:    "empty kubeconfig with invalid KUBECONFIG env",
			kubecfg: "",
			kubectx: "",
			setupEnv: func() {
				// Set KUBECONFIG to a non-existent file
				os.Setenv("KUBECONFIG", "/tmp/nonexistent-kubeconfig-for-test")
			},
			wantErr:     true,
			errContains: "invalid configuration",
		},
		{
			name:    "explicit context override with non-existent context",
			kubecfg: "",
			kubectx: "custom-context",
			setupEnv: func() {
				// Set KUBECONFIG to a non-existent file
				os.Setenv("KUBECONFIG", "/tmp/nonexistent-kubeconfig-for-test")
			},
			wantErr:     true,
			errContains: "context \"custom-context\" does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()

			_, err := DefaultK8s(tt.kubecfg, tt.kubectx)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DefaultK8s() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("DefaultK8s() error = %v, want error containing %v", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("DefaultK8s() unexpected error = %v", err)
			}
		})
	}
}

func TestEnablePsql17(t *testing.T) {
	// Save and restore the original paths.Data value.
	origData := paths.Data
	t.Cleanup(func() { paths.Data = origData })

	t.Run("non-existent pgdata directory returns true", func(t *testing.T) {
		paths.Data = filepath.Join(t.TempDir(), "does-not-exist")
		ok, err := EnablePsql17()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Fatal("expected true when pgdata directory does not exist")
		}
	})

	t.Run("permission denied on pgdata directory returns true", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("skipping permission test when running as root")
		}

		base := t.TempDir()
		pgDataDir := filepath.Join(base, paths.PvPsql, "pgdata")
		if err := os.MkdirAll(pgDataDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pgDataDir, "PG_VERSION"), []byte("16\n"), 0644); err != nil {
			t.Fatal(err)
		}
		// Remove all permissions from the pgdata directory.
		if err := os.Chmod(pgDataDir, 0000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(pgDataDir, 0755) })

		paths.Data = base
		ok, err := EnablePsql17()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Fatal("expected true when pgdata directory has permission denied")
		}
	})

	t.Run("version 17 returns true", func(t *testing.T) {
		base := t.TempDir()
		pgDataDir := filepath.Join(base, paths.PvPsql, "pgdata")
		if err := os.MkdirAll(pgDataDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pgDataDir, "PG_VERSION"), []byte("17\n"), 0644); err != nil {
			t.Fatal(err)
		}

		paths.Data = base
		ok, err := EnablePsql17()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Fatal("expected true when PG_VERSION is 17")
		}
	})

	t.Run("version 13 returns false", func(t *testing.T) {
		base := t.TempDir()
		pgDataDir := filepath.Join(base, paths.PvPsql, "pgdata")
		if err := os.MkdirAll(pgDataDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pgDataDir, "PG_VERSION"), []byte("13\n"), 0644); err != nil {
			t.Fatal(err)
		}

		paths.Data = base
		ok, err := EnablePsql17()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Fatal("expected false when PG_VERSION is 13")
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr || len(s) > len(substr) && contains(s[1:], substr)
}
