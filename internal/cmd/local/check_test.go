package local

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/airbytehq/abctl/internal/cmd/local/docker"
	"github.com/airbytehq/abctl/internal/cmd/local/docker/dockertest"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/go-connections/nat"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

func TestDockerInstalled(t *testing.T) {
	t.Cleanup(func() {
		dockerClient = nil
	})

	dockerClient = &docker.Docker{
		Client: dockertest.MockClient{
			FnServerVersion: func(ctx context.Context) (types.Version, error) {
				return types.Version{
					Platform: struct{ Name string }{Name: "test"},
					Version:  "version",
					Arch:     "arch",
				}, nil
			},
			FnInfo: func(ctx context.Context) (system.Info, error) {
				return system.Info{}, nil
			},
		},
	}

	tel := mockTelemetryClient{
		attr: func(key, val string) {},
	}

	version, err := dockerInstalled(context.Background(), &tel)
	if err != nil {
		t.Error("unexpected error:", err)
	}

	expectedVersion := docker.Version{
		Version:  "version",
		Arch:     "arch",
		Platform: "test",
	}

	if d := cmp.Diff(expectedVersion, version); d != "" {
		t.Error("unexpected version:", d)
	}
}

func TestDockerInstalled_Error(t *testing.T) {
	t.Cleanup(func() {
		dockerClient = nil
	})

	dockerClient = &docker.Docker{
		Client: dockertest.MockClient{
			FnServerVersion: func(ctx context.Context) (types.Version, error) {
				return types.Version{}, errors.New("test")
			},
		},
	}

	_, err := dockerInstalled(context.Background(), &mockTelemetryClient{})
	if err == nil {
		t.Error("unexpected error:", err)
	}
}

func TestPortAvailable_Available(t *testing.T) {
	// spin up a listener to find a port and then shut it down to ensure that port is available
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal("unable to create listener", err)
	}
	p := port(listener.Addr().String())
	if err := listener.Close(); err != nil {
		t.Fatal("unable to close listener", err)
	}

	err = portAvailable(context.Background(), p)
	if err != nil {
		t.Error("portAvailable returned unexpected error", err)
	}
}

func TestPortAvailable_Unavailable(t *testing.T) {
	// spin up a listener to find a port, and leave it running so that port is unavailable
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal("unable to create listener", err)
	}
	defer listener.Close()
	p := port(listener.Addr().String())

	err = portAvailable(context.Background(), p)
	// expecting an error
	if err == nil {
		t.Error("portAvailable should have returned an error")
	}
	if !errors.Is(err, localerr.ErrPort) {
		t.Error("error should be of type ErrPort")
	}
}

func TestGetPort_Found(t *testing.T) {
	t.Cleanup(func() {
		dockerClient = nil
	})

	dockerClient = &docker.Docker{
		Client: dockertest.MockClient{
			FnContainerInspect: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Status: "running",
						},
						HostConfig: &container.HostConfig{
							PortBindings: nat.PortMap{
								"tcp/80": {{
									HostIP:   "0.0.0.0",
									HostPort: "8000",
								}},
							},
						},
					},
				}, nil
			},
		},
	}

	port, err := getPort(context.Background(), "test")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}
	if port != 8000 {
		t.Errorf("expected 8000 but got %d", port)
	}
}

func TestGetPort_NotRunning(t *testing.T) {
	t.Cleanup(func() {
		dockerClient = nil
	})

	dockerClient = &docker.Docker{
		Client: dockertest.MockClient{
			FnContainerInspect: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Status: "stopped",
						},
						HostConfig: &container.HostConfig{
							PortBindings: nat.PortMap{
								"tcp/80": {{
									HostIP:   "0.0.0.0",
									HostPort: "8000",
								}},
							},
						},
					},
				}, nil
			},
		},
	}

	_, err := getPort(context.Background(), "test")

	if !errors.Is(err, ContainerNotRunningError{"test-control-plane", "stopped"}) {
		t.Errorf("expected container not running error but got %v", err)
	}
}

func TestGetPort_Missing(t *testing.T) {
	t.Cleanup(func() {
		dockerClient = nil
	})

	dockerClient = &docker.Docker{
		Client: dockertest.MockClient{
			FnContainerInspect: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Status: "running",
						},
						HostConfig: &container.HostConfig{
							PortBindings: nat.PortMap{
								"tcp/80": {{
									HostIP:   "1.2.3.4",
									HostPort: "8000",
								}},
							},
						},
					},
				}, nil
			},
		},
	}

	_, err := getPort(context.Background(), "test")
	if err == nil {
		t.Error("expected error")
	}
}

func TestGetPort_Invalid(t *testing.T) {
	t.Cleanup(func() {
		dockerClient = nil
	})

	dockerClient = &docker.Docker{
		Client: dockertest.MockClient{
			FnContainerInspect: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Status: "running",
						},
						HostConfig: &container.HostConfig{
							PortBindings: nat.PortMap{
								"tcp/80": {{
									HostIP:   "0.0.0.0",
									HostPort: "NaN",
								}},
							},
						},
					},
				}, nil
			},
		},
	}

	_, err := getPort(context.Background(), "test")
	var invalidPortErr InvalidPortError
	if !errors.As(err, &invalidPortErr) {
		t.Errorf("expected invalid port error but got %v", err)
	}
}

func TestGetPort_InpsectErr(t *testing.T) {
	t.Cleanup(func() {
		dockerClient = nil
	})

	dockerClient = &docker.Docker{
		Client: dockertest.MockClient{
			FnContainerInspect: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
				return types.ContainerJSON{}, errors.New("test err")
			},
		},
	}

	_, err := getPort(context.Background(), "test")
	if !errors.Is(err, ErrUnableToInspect) {
		t.Errorf("expected ErrUnableToInspect but got %v", err)
	}
}

// port returns the port from a string value in the format of "ipv4:port" or "ip::v6:port"
func port(s string) int {
	vals := strings.Split(s, ":")
	p, _ := strconv.Atoi(vals[len(vals)-1])
	return p
}

// --- mocks
var _ telemetry.Client = (*mockTelemetryClient)(nil)

type mockTelemetryClient struct {
	start   func(ctx context.Context, eventType telemetry.EventType) error
	success func(ctx context.Context, eventType telemetry.EventType) error
	failure func(ctx context.Context, eventType telemetry.EventType, err error) error
	attr    func(key, val string)
	user    func() uuid.UUID
	wrap    func(context.Context, telemetry.EventType, func() error) error
}

func (m *mockTelemetryClient) Start(ctx context.Context, eventType telemetry.EventType) error {
	return m.start(ctx, eventType)
}

func (m *mockTelemetryClient) Success(ctx context.Context, eventType telemetry.EventType) error {
	return m.success(ctx, eventType)
}

func (m *mockTelemetryClient) Failure(ctx context.Context, eventType telemetry.EventType, err error) error {
	return m.failure(ctx, eventType, err)
}

func (m *mockTelemetryClient) Attr(key, val string) {
	m.attr(key, val)
}

func (m *mockTelemetryClient) User() uuid.UUID {
	return m.user()
}

func (m *mockTelemetryClient) Wrap(ctx context.Context, et telemetry.EventType, f func() error) error {
	return m.wrap(ctx, et, f)
}
