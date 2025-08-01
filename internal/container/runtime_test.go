package container

import (
	"os"
	"strings"
	"testing"
)

func TestRuntimeString(t *testing.T) {
	tests := []struct {
		runtime Runtime
		want    string
	}{
		{Docker, "docker"},
		{Podman, "podman"},
		{Auto, "auto"},
		{Runtime(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.runtime.String(); got != tt.want {
				t.Errorf("Runtime.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Save original env vars
	origRuntime := os.Getenv("ABCTL_CONTAINER_RUNTIME")
	origContainerHost := os.Getenv("CONTAINER_HOST")
	origDockerHost := os.Getenv("DOCKER_HOST")
	origRootful := os.Getenv("ABCTL_PREFER_ROOTFUL")

	// Clean up after test
	defer func() {
		setEnvVar("ABCTL_CONTAINER_RUNTIME", origRuntime)
		setEnvVar("CONTAINER_HOST", origContainerHost)
		setEnvVar("DOCKER_HOST", origDockerHost)
		setEnvVar("ABCTL_PREFER_ROOTFUL", origRootful)
	}()

	tests := []struct {
		name        string
		envVars     map[string]string
		wantRuntime Runtime
		wantRootless bool
		wantSocket  string
	}{
		{
			name:        "default config",
			envVars:     map[string]string{},
			wantRuntime: Auto,
			wantRootless: true,
			wantSocket:  "",
		},
		{
			name: "docker runtime specified",
			envVars: map[string]string{
				"ABCTL_CONTAINER_RUNTIME": "docker",
			},
			wantRuntime: Docker,
			wantRootless: true,
			wantSocket:  "",
		},
		{
			name: "podman runtime specified",
			envVars: map[string]string{
				"ABCTL_CONTAINER_RUNTIME": "podman",
			},
			wantRuntime: Podman,
			wantRootless: true,
			wantSocket:  "",
		},
		{
			name: "container host specified",
			envVars: map[string]string{
				"CONTAINER_HOST": "unix:///custom/socket.sock",
			},
			wantRuntime: Auto,
			wantRootless: true,
			wantSocket:  "unix:///custom/socket.sock",
		},
		{
			name: "docker host specified",
			envVars: map[string]string{
				"DOCKER_HOST": "unix:///docker/socket.sock",
			},
			wantRuntime: Auto,
			wantRootless: true,
			wantSocket:  "unix:///docker/socket.sock",
		},
		{
			name: "prefer rootful",
			envVars: map[string]string{
				"ABCTL_PREFER_ROOTFUL": "1",
			},
			wantRuntime: Auto,
			wantRootless: false,
			wantSocket:  "",
		},
		{
			name: "container host overrides docker host",
			envVars: map[string]string{
				"CONTAINER_HOST": "unix:///container/socket.sock",
				"DOCKER_HOST":    "unix:///docker/socket.sock",
			},
			wantRuntime: Auto,
			wantRootless: true,
			wantSocket:  "unix:///container/socket.sock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars first
			setEnvVar("ABCTL_CONTAINER_RUNTIME", "")
			setEnvVar("CONTAINER_HOST", "")
			setEnvVar("DOCKER_HOST", "")
			setEnvVar("ABCTL_PREFER_ROOTFUL", "")

			// Set test env vars
			for k, v := range tt.envVars {
				setEnvVar(k, v)
			}

			config := LoadConfig()

			if config.Runtime != tt.wantRuntime {
				t.Errorf("LoadConfig().Runtime = %v, want %v", config.Runtime, tt.wantRuntime)
			}
			if config.PreferRootless != tt.wantRootless {
				t.Errorf("LoadConfig().PreferRootless = %v, want %v", config.PreferRootless, tt.wantRootless)
			}
			if config.SocketPath != tt.wantSocket {
				t.Errorf("LoadConfig().SocketPath = %v, want %v", config.SocketPath, tt.wantSocket)
			}
		})
	}
}

// setEnvVar sets an environment variable, handling empty strings
func setEnvVar(key, value string) {
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
}

func TestVersion(t *testing.T) {
	version := Version{
		Version:  "1.0.0",
		Arch:     "amd64",
		Platform: "linux",
		Runtime:  "docker",
	}

	// Test that all fields are set correctly
	if version.Version != "1.0.0" {
		t.Errorf("Version.Version = %v, want %v", version.Version, "1.0.0")
	}
	if version.Arch != "amd64" {
		t.Errorf("Version.Arch = %v, want %v", version.Arch, "amd64")
	}
	if version.Platform != "linux" {
		t.Errorf("Version.Platform = %v, want %v", version.Platform, "linux")
	}
	if version.Runtime != "docker" {
		t.Errorf("Version.Runtime = %v, want %v", version.Runtime, "docker")
	}
}

func TestConfigRuntimeParsing(t *testing.T) {
	tests := []struct {
		input string
		want  Runtime
	}{
		{"docker", Docker},
		{"DOCKER", Docker},
		{"Docker", Docker},
		{"podman", Podman},
		{"PODMAN", Podman},
		{"Podman", Podman},
		{"unknown", Auto}, // defaults to Auto for unknown values
		{"", Auto},        // defaults to Auto for empty values
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			os.Setenv("ABCTL_CONTAINER_RUNTIME", tt.input)
			defer os.Unsetenv("ABCTL_CONTAINER_RUNTIME")

			config := LoadConfig()
			if config.Runtime != tt.want {
				t.Errorf("LoadConfig() with ABCTL_CONTAINER_RUNTIME=%q, Runtime = %v, want %v", tt.input, config.Runtime, tt.want)
			}
		})
	}
}