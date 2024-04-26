package local

import (
	"context"
	"errors"
	"github.com/airbytehq/abctl/internal/local/docker"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/docker/docker/api/types"
	"github.com/google/go-cmp/cmp"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestDockerInstalled(t *testing.T) {
	t.Cleanup(func() {
		dockerClient = nil
	})

	dockerClient = &docker.Docker{
		Client: mockDockerClient{
			serverVersion: func(ctx context.Context) (types.Version, error) {
				return types.Version{
					Platform: struct{ Name string }{Name: "test"},
					Version:  "version",
					Arch:     "arch",
				}, nil
			},
		},
	}

	err := dockerInstalled(context.Background(), telemetry.NoopClient{})
	if err != nil {
		t.Error("unexpected error:", err)
	}
}

func TestDockerInstalled_TelemetryAttrs(t *testing.T) {
	t.Cleanup(func() {
		dockerClient = nil
	})

	platformName := "platform name"
	version := "version"
	arch := "arch"

	dockerClient = &docker.Docker{
		Client: mockDockerClient{
			serverVersion: func(ctx context.Context) (types.Version, error) {
				return types.Version{
					Platform: struct{ Name string }{Name: platformName},
					Version:  version,
					Arch:     arch,
				}, nil
			},
		},
	}

	attrs := map[string]string{}
	telemetryClient := &mockTelemetryClient{attr: func(key, val string) {
		attrs[key] = val
	}}

	err := dockerInstalled(context.Background(), telemetryClient, os.TempDir())
	if err != nil {
		t.Error("unexpected error:", err)
	}
	expAttrs := map[string]string{
		"docker_version":  version,
		"docker_arch":     arch,
		"docker_platform": platformName,
	}
	if d := cmp.Diff(expAttrs, attrs); d != "" {
		t.Error("mismatched attributes:", d)
	}
}

func TestDockerInstalled_Error(t *testing.T) {
	t.Cleanup(func() {
		dockerClient = nil
	})

	dockerClient = &docker.Docker{
		Client: mockDockerClient{
			serverVersion: func(ctx context.Context) (types.Version, error) {
				return types.Version{}, errors.New("test")
			},
		},
	}

	err := dockerInstalled(context.Background(), telemetry.NoopClient{}, os.TempDir())
	if err == nil {
		t.Error("unexpected error:", err)
	}
}

func TestPortAvailable_Available(t *testing.T) {
	// spin up a listener to find a port and then shut it down to ensure that port is available
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal("could not create listener", err)
	}
	p := port(listener.Addr().String())
	if err := listener.Close(); err != nil {
		t.Fatal("could not close listener", err)
	}

	err = portAvailable(context.Background(), p)
	if err != nil {
		t.Error("portAvailable returned unexpected error", err)
	}
}

func TestPortAvailable_Unavailable(t *testing.T) {
	// spin up a listener to find a port, and leave it running so that port is unavailable
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal("could not create listener", err)
	}
	defer listener.Close()
	p := port(listener.Addr().String())

	err = portAvailable(context.Background(), p)
	// expecting an error
	if err == nil {
		t.Error("portAvailable returned nil error")
	}
}

// port returns the port from a string value in the format of "ipv4:port" or "ip::v6:port"
func port(s string) int {
	vals := strings.Split(s, ":")
	p, _ := strconv.Atoi(vals[len(vals)-1])
	return p
}

// mocks

var _ docker.Client = (*mockDockerClient)(nil)

type mockDockerClient struct {
	serverVersion    func(ctx context.Context) (types.Version, error)
	containerInspect func(ctx context.Context, containerID string) (types.ContainerJSON, error)
}

func (m mockDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return m.containerInspect(ctx, containerID)
}

func (m mockDockerClient) ServerVersion(ctx context.Context) (types.Version, error) {
	return m.serverVersion(ctx)
}

var _ telemetry.Client = (*mockTelemetryClient)(nil)

type mockTelemetryClient struct {
	start   func(eventType telemetry.EventType) error
	success func(eventType telemetry.EventType) error
	failure func(eventType telemetry.EventType, err error) error
	attr    func(key, val string)
}

func (m *mockTelemetryClient) Start(eventType telemetry.EventType) error {
	return m.start(eventType)
}

func (m *mockTelemetryClient) Success(eventType telemetry.EventType) error {
	return m.success(eventType)
}

func (m *mockTelemetryClient) Failure(eventType telemetry.EventType, err error) error {
	return m.failure(eventType, err)
}

func (m *mockTelemetryClient) Attr(key, val string) {
	m.attr(key, val)
}
