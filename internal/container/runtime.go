package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Runtime represents the container runtime type
type Runtime int

const (
	// Docker runtime
	Docker Runtime = iota
	// Podman runtime
	Podman
	// Auto automatically detects the available runtime
	Auto
)

func (r Runtime) String() string {
	switch r {
	case Docker:
		return "docker"
	case Podman:
		return "podman"
	case Auto:
		return "auto"
	default:
		return "unknown"
	}
}

// Client interface for container operations. This matches the existing docker.Client interface
// but is now runtime-agnostic.
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

// SocketDetector detects container runtime socket paths
type SocketDetector interface {
	DetectSockets(ctx context.Context, goos string) ([]string, error)
}

// ClientFactory creates container runtime clients
type ClientFactory interface {
	CreateClient(ctx context.Context, runtime Runtime) (Client, error)
}

// Config contains container runtime configuration
type Config struct {
	Runtime        Runtime
	PreferRootless bool
	SocketPath     string
}

// LoadConfig loads container runtime configuration from environment
func LoadConfig() *Config {
	cfg := &Config{
		Runtime:        Auto,
		PreferRootless: true,
	}

	// Check environment variables - support KIND's convention too
	runtimeEnv := os.Getenv("ABCTL_CONTAINER_RUNTIME")
	if runtimeEnv == "" {
		// Also check KIND's environment variable for compatibility
		runtimeEnv = os.Getenv("KIND_EXPERIMENTAL_PROVIDER")
	}
	
	if runtimeEnv != "" {
		switch strings.ToLower(runtimeEnv) {
		case "docker":
			cfg.Runtime = Docker
		case "podman":
			cfg.Runtime = Podman
		}
	}

	if socket := os.Getenv("CONTAINER_HOST"); socket != "" {
		cfg.SocketPath = socket
	} else if socket := os.Getenv("DOCKER_HOST"); socket != "" {
		cfg.SocketPath = socket
	}

	if os.Getenv("ABCTL_PREFER_ROOTFUL") != "" {
		cfg.PreferRootless = false
	}

	return cfg
}

// Version contains version information for a container runtime
type Version struct {
	Version  string
	Arch     string
	Platform string
	Runtime  string
}

// ContainerRuntime wraps a container client with runtime information
type ContainerRuntime struct {
	Client   Client
	Type     Runtime
	config   *Config
	provider *Provider
}

// Version returns version information from the container runtime
func (cr *ContainerRuntime) Version(ctx context.Context) (Version, error) {
	ver, err := cr.Client.ServerVersion(ctx)
	if err != nil {
		return Version{}, fmt.Errorf("unable to determine server version: %w", err)
	}

	return Version{
		Version:  ver.Version,
		Arch:     ver.Arch,
		Platform: ver.Platform.Name,
		Runtime:  cr.Type.String(),
	}, nil
}

// IsRootless returns true if the runtime is running in rootless mode
func (cr *ContainerRuntime) IsRootless() bool {
	// If we have a provider, use its rootless detection
	if cr.provider != nil {
		if rootless, err := cr.provider.IsRootless(context.Background()); err == nil {
			return rootless
		}
	}
	// Fall back to configuration-based detection
	return cr.config.PreferRootless && os.Getuid() != 0
}

// Executor returns the command executor for this runtime
func (cr *ContainerRuntime) Executor() CommandExecutor {
	if cr.provider != nil {
		return cr.provider.Executor
	}
	return nil
}