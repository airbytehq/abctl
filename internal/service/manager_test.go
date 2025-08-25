package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultK8s_KubeconfigHandling(t *testing.T) {
	// Save original KUBECONFIG env var
	originalKubeconfig := os.Getenv("KUBECONFIG")
	defer func() {
		os.Setenv("KUBECONFIG", originalKubeconfig)
	}()

	tests := []struct {
		name       string
		kubecfg    string
		kubectx    string
		setupEnv   func()
		wantErr    bool
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

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr || len(s) > len(substr) && contains(s[1:], substr)
}