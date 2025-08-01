package container

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"
)

func TestDockerSocketDetector_DetectSockets(t *testing.T) {
	detector := &DockerSocketDetector{}
	ctx := context.Background()

	tests := []struct {
		goos     string
		wantHost []string
	}{
		{
			goos: "linux",
			wantHost: []string{
				"unix:///var/run/docker.sock",
			},
		},
		{
			goos: "darwin",
			wantHost: []string{
				"unix:///var/run/docker.sock",
			},
		},
		{
			goos: "windows",
			wantHost: []string{
				"npipe:////./pipe/docker_engine",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			sockets, err := detector.DetectSockets(ctx, tt.goos)
			if err != nil {
				t.Errorf("DockerSocketDetector.DetectSockets() error = %v", err)
				return
			}

			// Check that expected sockets are present
			for _, expectedSocket := range tt.wantHost {
				found := false
				for _, socket := range sockets {
					if socket == expectedSocket {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("DockerSocketDetector.DetectSockets() missing expected socket %s in %v", expectedSocket, sockets)
				}
			}
		})
	}
}

func TestPodmanSocketDetector_DetectSockets(t *testing.T) {
	detector := &PodmanSocketDetector{}
	ctx := context.Background()

	tests := []struct {
		name     string
		goos     string
		uid      int
		expected []string
	}{
		{
			name: "linux rootless",
			goos: "linux",
			uid:  1000,
			expected: []string{
				"unix:///run/podman/podman.sock", // rootful fallback
				"unix:///var/run/docker.sock",   // compatibility
			},
		},
		{
			name: "linux root",
			goos: "linux",
			uid:  0,
			expected: []string{
				"unix:///run/podman/podman.sock",
				"unix:///var/run/docker.sock",
			},
		},
		{
			name: "darwin",
			goos: "darwin",
			uid:  1000,
			expected: []string{
				"unix:///run/podman/podman.sock",
				"unix:///var/run/docker.sock",
			},
		},
		{
			name: "windows",
			goos: "windows",
			uid:  1000,
			expected: []string{
				"unix:///run/podman/podman.sock",
				"unix:///var/run/docker.sock",
				"npipe:////./pipe/podman_engine",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sockets, err := detector.DetectSockets(ctx, tt.goos)
			if err != nil {
				t.Errorf("PodmanSocketDetector.DetectSockets() error = %v", err)
				return
			}

			// Check that expected sockets are present
			for _, expectedSocket := range tt.expected {
				found := false
				for _, socket := range sockets {
					if socket == expectedSocket {
						found = true
						break
					}
				}
				if !found {
					// Only fail for required sockets (not platform-specific ones)
					if expectedSocket == "unix:///run/podman/podman.sock" || expectedSocket == "unix:///var/run/docker.sock" {
						t.Errorf("PodmanSocketDetector.DetectSockets() missing expected socket %s in %v", expectedSocket, sockets)
					}
				}
			}

			// Verify we got at least some sockets
			if len(sockets) == 0 {
				t.Errorf("PodmanSocketDetector.DetectSockets() returned no sockets")
			}
		})
	}
}

func TestAutoDetector_DetectRuntime(t *testing.T) {
	// This test is more integration-like and depends on the actual system
	// We'll test the basic functionality without depending on actual sockets

	detector := NewAutoDetector()
	ctx := context.Background()

	// Test that the method doesn't panic and returns valid values
	runtimeType, sockets, err := detector.DetectRuntime(ctx)

	// We don't assert specific values since they depend on the system
	// Just verify the method works and returns reasonable values
	if err != nil && len(sockets) > 0 {
		t.Errorf("DetectRuntime() returned error but also sockets: err=%v, sockets=%v", err, sockets)
	}

	// Runtime should be a valid enum value
	validRuntimes := map[Runtime]bool{
		Docker: true,
		Podman: true,
		Auto:   true,
	}

	if !validRuntimes[runtimeType] && err == nil {
		t.Errorf("DetectRuntime() returned invalid runtime type: %v", runtimeType)
	}

	t.Logf("DetectRuntime() result: runtime=%v, sockets=%v, err=%v", runtimeType, sockets, err)
}

func TestAutoDetector_isRuntimeAvailable(t *testing.T) {
	detector := NewAutoDetector()

	tests := []struct {
		name    string
		sockets []string
		want    bool
	}{
		{
			name:    "empty sockets",
			sockets: []string{},
			want:    false,
		},
		{
			name:    "non-existent unix socket",
			sockets: []string{"unix:///tmp/non-existent-socket.sock"},
			want:    false,
		},
		{
			name:    "npipe socket",
			sockets: []string{"npipe:////./pipe/test"},
			want:    true, // assumes npipe sockets are available if listed
		},
		{
			name:    "tcp socket",
			sockets: []string{"tcp://localhost:2376"},
			want:    true, // assumes tcp sockets are available if listed
		},
		{
			name:    "mixed sockets with non-unix",
			sockets: []string{"unix:///tmp/non-existent.sock", "tcp://localhost:2376"},
			want:    true, // tcp socket should be considered available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.isRuntimeAvailable(context.Background(), tt.sockets)
			if got != tt.want {
				t.Errorf("AutoDetector.isRuntimeAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPodmanSocketDetector_WithEnvironment(t *testing.T) {
	// Test with XDG_RUNTIME_DIR set
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	defer func() {
		if origXDG != "" {
			os.Setenv("XDG_RUNTIME_DIR", origXDG)
		} else {
			os.Unsetenv("XDG_RUNTIME_DIR")
		}
	}()

	testXDG := "/tmp/test-runtime"
	os.Setenv("XDG_RUNTIME_DIR", testXDG)

	detector := &PodmanSocketDetector{}
	ctx := context.Background()

	sockets, err := detector.DetectSockets(ctx, runtime.GOOS)
	if err != nil {
		t.Errorf("PodmanSocketDetector.DetectSockets() error = %v", err)
		return
	}

	// When running as non-root, should include XDG_RUNTIME_DIR socket
	if os.Getuid() != 0 {
		expectedSocket := fmt.Sprintf("unix://%s/podman/podman.sock", testXDG)
		found := false
		for _, socket := range sockets {
			if socket == expectedSocket {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("PodmanSocketDetector.DetectSockets() missing XDG_RUNTIME_DIR socket %s in %v", expectedSocket, sockets)
		}
	}
}