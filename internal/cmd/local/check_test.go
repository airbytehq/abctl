package local

import (
	"context"
	"errors"
	"github.com/airbytehq/abctl/internal/cmd/local/docker"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/docker/docker/api/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"net"
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

	version, err := dockerInstalled(context.Background())
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
		Client: mockDockerClient{
			serverVersion: func(ctx context.Context) (types.Version, error) {
				return types.Version{}, errors.New("test")
			},
		},
	}

	_, err := dockerInstalled(context.Background())
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
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal("could not create listener", err)
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
	start   func(ctx context.Context, eventType telemetry.EventType) error
	success func(ctx context.Context, eventType telemetry.EventType) error
	failure func(ctx context.Context, eventType telemetry.EventType, err error) error
	attr    func(key, val string)
	user    func() uuid.UUID
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
