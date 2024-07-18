package local

import (
	"context"
	"errors"
	"github.com/airbytehq/abctl/internal/cmd/local/docker"
	"github.com/airbytehq/abctl/internal/cmd/local/dockertest"
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
		Client: dockertest.MockClient{
			FnServerVersion: func(ctx context.Context) (types.Version, error) {
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
		Client: dockertest.MockClient{
			FnServerVersion: func(ctx context.Context) (types.Version, error) {
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

// port returns the port from a string value in the format of "ipv4:port" or "ip::v6:port"
func port(s string) int {
	vals := strings.Split(s, ":")
	p, _ := strconv.Atoi(vals[len(vals)-1])
	return p
}

//// mocks
//
//var _ docker.Client = (*mockDockerClient)(nil)
//
//type mockDockerClient struct {
//	containerCreate   func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error)
//	containerInspect  func(ctx context.Context, containerID string) (types.ContainerJSON, error)
//	containerRemove   func(ctx context.Context, container string, options container.RemoveOptions) error
//	containerStart    func(ctx context.Context, container string, options container.StartOptions) error
//	containerStop     func(ctx context.Context, container string, options container.StopOptions) error
//	copyFromContainer func(ctx context.Context, container, srcPath string) (io.ReadCloser, types.ContainerPathStat, error)
//
//	containerExecCreate  func(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error)
//	containerExecInspect func(ctx context.Context, execID string) (types.ContainerExecInspect, error)
//	containerExecStart   func(ctx context.Context, execID string, config types.ExecStartCheck) error
//
//	imageList func(ctx context.Context, options image.ListOptions) ([]image.Summary, error)
//	imagePull func(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error)
//
//	serverVersion func(ctx context.Context) (types.Version, error)
//	volumeInspect func(ctx context.Context, volumeID string) (volume.Volume, error)
//}
//
//func (m mockDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error) {
//	return m.containerCreate(ctx, config, hostConfig, networkingConfig, platform, containerName)
//}
//
//func (m mockDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
//	return m.containerInspect(ctx, containerID)
//}
//
//func (m mockDockerClient) ContainerRemove(ctx context.Context, container string, options container.RemoveOptions) error {
//	return m.containerRemove(ctx, container, options)
//}
//
//func (m mockDockerClient) ContainerStart(ctx context.Context, container string, options container.StartOptions) error {
//	return m.containerStart(ctx, container, options)
//}
//
//func (m mockDockerClient) ContainerStop(ctx context.Context, container string, options container.StopOptions) error {
//	return m.containerStop(ctx, container, options)
//}
//
//func (m mockDockerClient) CopyFromContainer(ctx context.Context, container, srcPath string) (io.ReadCloser, types.ContainerPathStat, error) {
//	return m.copyFromContainer(ctx, container, srcPath)
//}
//
//func (m mockDockerClient) ContainerExecCreate(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error) {
//	return m.containerExecCreate(ctx, container, config)
//}
//
//func (m mockDockerClient) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
//	return m.containerExecInspect(ctx, execID)
//}
//
//func (m mockDockerClient) ContainerExecStart(ctx context.Context, execID string, config types.ExecStartCheck) error {
//	return m.containerExecStart(ctx, execID, config)
//}
//
//func (m mockDockerClient) ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
//	// by default return a list with one (empty) item in it
//	if m.imageList == nil {
//		return []image.Summary{{}}, nil
//	}
//	return m.imageList(ctx, options)
//}
//
//func (m mockDockerClient) ImagePull(ctx context.Context, img string, options image.PullOptions) (io.ReadCloser, error) {
//	// by default return a nop closer (with an empty string)
//	if m.imagePull == nil {
//		return io.NopCloser(strings.NewReader("")), nil
//	}
//	return m.imagePull(ctx, img, options)
//}
//
//func (m mockDockerClient) ServerVersion(ctx context.Context) (types.Version, error) {
//	return m.serverVersion(ctx)
//}
//
//func (m mockDockerClient) VolumeInspect(ctx context.Context, volumeID string) (volume.Volume, error) {
//	return m.volumeInspect(ctx, volumeID)
//}

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
