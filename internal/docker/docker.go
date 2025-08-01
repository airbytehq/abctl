package docker

import (
	"context"
	"fmt"
	"io"

	containerruntime "github.com/airbytehq/abctl/internal/container"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Version contains al the version information that is being tracked.
// Deprecated: Use containerruntime.Version instead
type Version struct {
	// Version is the platform version
	Version string
	// Arch is the platform architecture
	Arch string
	// Platform is the platform name
	Platform string
}

// Client interface for testing purposes. Includes only the methods used by the underlying docker package.
// Deprecated: Use containerruntime.Client instead
type Client interface {
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error)
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ContainerRemove(ctx context.Context, container string, options container.RemoveOptions) error
	ContainerStart(ctx context.Context, container string, options container.StartOptions) error
	ContainerStop(ctx context.Context, container string, options container.StopOptions) error
	CopyFromContainer(ctx context.Context, container, srcPath string) (io.ReadCloser, container.PathStat, error)

	ContainerExecCreate(ctx context.Context, container string, config container.ExecOptions) (types.IDResponse, error)
	ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error)
	ContainerExecStart(ctx context.Context, execID string, config container.ExecStartOptions) error

	ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error)
	ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error)
	ImageSave(ctx context.Context, imageIDs []string) (io.ReadCloser, error)

	ServerVersion(ctx context.Context) (types.Version, error)
	VolumeInspect(ctx context.Context, volumeID string) (volume.Volume, error)
	Info(ctx context.Context) (system.Info, error)
}

// Docker for handling communication with container runtimes.
// Can be created with default settings by calling New or with a custom Client by manually instantiating this type.
// Deprecated: Use containerruntime.ContainerRuntime instead for new code
type Docker struct {
	Client Client
	runtime *containerruntime.ContainerRuntime
}

// New returns a new Docker type with a default Client implementation.
// This now uses the container runtime abstraction and supports both Docker and Podman.
func New(ctx context.Context) (*Docker, error) {
	runtime, err := containerruntime.New(ctx)
	if err != nil {
		return nil, err
	}

	return &Docker{
		Client:  runtime.Client,
		runtime: runtime,
	}, nil
}

// NewWithConfig creates a new Docker instance with specific container runtime configuration
func NewWithConfig(ctx context.Context, config *containerruntime.Config) (*Docker, error) {
	runtime, err := containerruntime.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	return &Docker{
		Client:  runtime.Client,
		runtime: runtime,
	}, nil
}

// Version returns the version information from the underlying container runtime.
func (d *Docker) Version(ctx context.Context) (Version, error) {
	if d.runtime != nil {
		ver, err := d.runtime.Version(ctx)
		if err != nil {
			return Version{}, err
		}
		return Version{
			Version:  ver.Version,
			Arch:     ver.Arch,
			Platform: ver.Platform,
		}, nil
	}

	// Fallback for backward compatibility
	ver, err := d.Client.ServerVersion(ctx)
	if err != nil {
		return Version{}, fmt.Errorf("unable to determine server version: %w", err)
	}

	return Version{
		Version:  ver.Version,
		Arch:     ver.Arch,
		Platform: ver.Platform.Name,
	}, nil
}

// RuntimeType returns the type of container runtime being used
func (d *Docker) RuntimeType() containerruntime.Runtime {
	if d.runtime != nil {
		return d.runtime.Type
	}
	return containerruntime.Docker // Default assumption for backward compatibility
}

// IsRootless returns true if the container runtime is running in rootless mode
func (d *Docker) IsRootless() bool {
	if d.runtime != nil {
		return d.runtime.IsRootless()
	}
	return false // Conservative default
}
