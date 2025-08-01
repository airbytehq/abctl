package container

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// RuntimeClient wraps a Docker client and provides runtime information
type RuntimeClient struct {
	client      *client.Client
	runtimeType string
}

// Ensure RuntimeClient implements the Client interface
var _ Client = (*RuntimeClient)(nil)

// ContainerCreate creates a new container
func (rc *RuntimeClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error) {
	return rc.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, containerName)
}

// ContainerInspect returns detailed information about a container
func (rc *RuntimeClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return rc.client.ContainerInspect(ctx, containerID)
}

// ContainerRemove removes a container
func (rc *RuntimeClient) ContainerRemove(ctx context.Context, container string, options container.RemoveOptions) error {
	return rc.client.ContainerRemove(ctx, container, options)
}

// ContainerStart starts a container
func (rc *RuntimeClient) ContainerStart(ctx context.Context, container string, options container.StartOptions) error {
	return rc.client.ContainerStart(ctx, container, options)
}

// ContainerStop stops a container
func (rc *RuntimeClient) ContainerStop(ctx context.Context, container string, options container.StopOptions) error {
	return rc.client.ContainerStop(ctx, container, options)
}

// CopyFromContainer copies files/folders from a container
func (rc *RuntimeClient) CopyFromContainer(ctx context.Context, container, srcPath string) (io.ReadCloser, container.PathStat, error) {
	return rc.client.CopyFromContainer(ctx, container, srcPath)
}

// ContainerExecCreate creates an exec instance
func (rc *RuntimeClient) ContainerExecCreate(ctx context.Context, container string, config container.ExecOptions) (types.IDResponse, error) {
	return rc.client.ContainerExecCreate(ctx, container, config)
}

// ContainerExecInspect inspects an exec instance
func (rc *RuntimeClient) ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error) {
	return rc.client.ContainerExecInspect(ctx, execID)
}

// ContainerExecStart starts an exec instance
func (rc *RuntimeClient) ContainerExecStart(ctx context.Context, execID string, config container.ExecStartOptions) error {
	return rc.client.ContainerExecStart(ctx, execID, config)
}

// ImageList lists images
func (rc *RuntimeClient) ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	return rc.client.ImageList(ctx, options)
}

// ImagePull pulls an image
func (rc *RuntimeClient) ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error) {
	return rc.client.ImagePull(ctx, refStr, options)
}

// ImageSave saves images to a tar archive
func (rc *RuntimeClient) ImageSave(ctx context.Context, imageIDs []string) (io.ReadCloser, error) {
	return rc.client.ImageSave(ctx, imageIDs)
}

// ServerVersion returns server version information
func (rc *RuntimeClient) ServerVersion(ctx context.Context) (types.Version, error) {
	return rc.client.ServerVersion(ctx)
}

// VolumeInspect inspects a volume
func (rc *RuntimeClient) VolumeInspect(ctx context.Context, volumeID string) (volume.Volume, error) {
	return rc.client.VolumeInspect(ctx, volumeID)
}

// Info returns system information
func (rc *RuntimeClient) Info(ctx context.Context) (system.Info, error) {
	return rc.client.Info(ctx)
}

// Close closes the underlying client connection
func (rc *RuntimeClient) Close() error {
	return rc.client.Close()
}

// RuntimeType returns the type of runtime this client is connected to
func (rc *RuntimeClient) RuntimeType() string {
	return rc.runtimeType
}