package local

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/docker"
	"github.com/airbytehq/abctl/internal/docker/dockertest"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/go-connections/nat"
	"github.com/google/go-cmp/cmp"
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

	tel := telemetry.MockClient{}

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

	_, err := dockerInstalled(context.Background(), &telemetry.MockClient{})
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

func TestValidateHostFlag(t *testing.T) {
	expectErr := func(host string, expect error) {
		err := validateHostFlag(host)
		if !errors.Is(err, expect) {
			t.Errorf("expected error %v for host %q but got %v", expect, host, err)
		}
	}
	expectErr("1.2.3.4", localerr.ErrIpAddressForHostFlag)
	expectErr("1.2.3.4:8000", localerr.ErrInvalidHostFlag)
	expectErr("1.2.3.4:8000", localerr.ErrInvalidHostFlag)
	expectErr("ABC-DEF-GHI.abcd.efgh", localerr.ErrInvalidHostFlag)
	expectErr("http://airbyte.foo-data-platform-sbx.bar.cloud", localerr.ErrInvalidHostFlag)

	expectOk := func(host string) {
		err := validateHostFlag(host)
		if err != nil {
			t.Errorf("unexpected error for host %q: %s", host, err)
		}
	}
	expectOk("foo")
	expectOk("foo.bar")
	expectOk("example.com")
	expectOk("sub.example01.com")
}

// port returns the port from a string value in the format of "ipv4:port" or "ip::v6:port"
func port(s string) int {
	vals := strings.Split(s, ":")
	p, _ := strconv.Atoi(vals[len(vals)-1])
	return p
}
