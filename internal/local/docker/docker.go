package docker

import (
	"context"
	"errors"
	"fmt"
	"github.com/airbytehq/abctl/internal/local/localerr"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"os"
	"runtime"
	"strconv"
)

// Version contains al the version information that is being tracked.
type Version struct {
	// Version is the platform version
	Version string
	// Arch is the platform architecture
	Arch string
	// Platform is the platform name
	Platform string
}

// Client interface for testing purposes. Includes only the methods used by the underlying docker package.
type Client interface {
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ServerVersion(context.Context) (types.Version, error)
}

// Docker for handling communication with the docker processes.
// Can be created with default settings by calling New or with a custom Client by manually
// instantiating this type.
type Docker struct {
	Client Client
}

// New returns a new Docker type with a default Client implementation.
func New() (*Docker, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine user home directory: %w", err)
	}

	var dockerCli *client.Client
	dockerOpts := []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}

	switch runtime.GOOS {
	case "darwin":
		// on mac, sometimes the docker host isn't set correctly, if it fails check the home directory
		dockerCli, err = client.NewClientWithOpts(append(dockerOpts, client.WithHost("unix:///var/run/docker.sock"))...)
		if err != nil {
			// keep the original error, as we'll join with the next error (if another error occurs)
			outerErr := err
			// this works as the last WithHost call will win
			dockerCli, err = client.NewClientWithOpts(append(dockerOpts, client.WithHost(fmt.Sprintf("unix:///%s/.docker/run/docker.sock", userHome)))...)
			if err != nil {
				err = fmt.Errorf("%w: %w", err, outerErr)
			}
		}
	case "windows":
		dockerCli, err = client.NewClientWithOpts(append(dockerOpts, client.WithHost("npipe:////./pipe/docker_engine"))...)
	default:
		dockerCli, err = client.NewClientWithOpts(append(dockerOpts, client.WithHost("unix:///var/run/docker.sock"))...)
	}

	if err != nil {
		return nil, fmt.Errorf("%w: could not create docker client: %w", localerr.ErrDocker, err)
	}

	return &Docker{Client: dockerCli}, nil
}

// Version returns the version information from the underlying docker process.
func (d *Docker) Version(ctx context.Context) (Version, error) {
	ver, err := d.Client.ServerVersion(ctx)
	if err != nil {
		return Version{}, fmt.Errorf("could not determine server version: %w", err)
	}

	return Version{
		Version:  ver.Version,
		Arch:     ver.Arch,
		Platform: ver.Platform.Name,
	}, nil

}

// Port returns the host-port the underlying docker process is currently bound to, for the given container.
// It determines this by walking through all the ports on the container and finding the one that is bound to ip 0.0.0.0.
func (d *Docker) Port(ctx context.Context, container string) (int, error) {
	ci, err := d.Client.ContainerInspect(ctx, container)
	if err != nil {
		return 0, fmt.Errorf("could not inspect container: %w", err)
	}

	for _, bindings := range ci.NetworkSettings.Ports {
		for _, ipPort := range bindings {
			if ipPort.HostIP == "0.0.0.0" {
				port, err := strconv.Atoi(ipPort.HostPort)
				if err != nil {
					return 0, fmt.Errorf("could not convert host port %s to integer: %w", ipPort.HostPort, err)
				}
				return port, nil
			}
		}
	}

	return 0, errors.New("could not determine port for container")
}
