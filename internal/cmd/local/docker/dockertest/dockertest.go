package dockertest

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type MockClient struct {
	FnContainerCreate      func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error)
	FnContainerInspect     func(ctx context.Context, containerID string) (types.ContainerJSON, error)
	FnContainerRemove      func(ctx context.Context, container string, options container.RemoveOptions) error
	FnContainerStart       func(ctx context.Context, container string, options container.StartOptions) error
	FnContainerStop        func(ctx context.Context, container string, options container.StopOptions) error
	FnCopyFromContainer    func(ctx context.Context, container, srcPath string) (io.ReadCloser, container.PathStat, error)
	FnContainerExecCreate  func(ctx context.Context, container string, config container.ExecOptions) (types.IDResponse, error)
	FnContainerExecInspect func(ctx context.Context, execID string) (container.ExecInspect, error)
	FnContainerExecStart   func(ctx context.Context, execID string, config container.ExecStartOptions) error
	FnImageList            func(ctx context.Context, options image.ListOptions) ([]image.Summary, error)
	FnImagePull            func(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error)
	FnServerVersion        func(ctx context.Context) (types.Version, error)
	FnVolumeInspect        func(ctx context.Context, volumeID string) (volume.Volume, error)
	FnInfo                   func(ctx context.Context) (system.Info, error)
}

func (m MockClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error) {
	return m.FnContainerCreate(ctx, config, hostConfig, networkingConfig, platform, containerName)
}

func (m MockClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return m.FnContainerInspect(ctx, containerID)
}

func (m MockClient) ContainerRemove(ctx context.Context, container string, options container.RemoveOptions) error {
	return m.FnContainerRemove(ctx, container, options)
}

func (m MockClient) ContainerStart(ctx context.Context, container string, options container.StartOptions) error {
	return m.FnContainerStart(ctx, container, options)
}

func (m MockClient) ContainerStop(ctx context.Context, container string, options container.StopOptions) error {
	return m.FnContainerStop(ctx, container, options)
}

func (m MockClient) CopyFromContainer(ctx context.Context, container, srcPath string) (io.ReadCloser, container.PathStat, error) {
	return m.FnCopyFromContainer(ctx, container, srcPath)
}

func (m MockClient) ContainerExecCreate(ctx context.Context, container string, config container.ExecOptions) (types.IDResponse, error) {
	return m.FnContainerExecCreate(ctx, container, config)
}

func (m MockClient) ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error) {
	return m.FnContainerExecInspect(ctx, execID)
}

func (m MockClient) ContainerExecStart(ctx context.Context, execID string, config container.ExecStartOptions) error {
	return m.FnContainerExecStart(ctx, execID, config)
}

func (m MockClient) ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	return m.FnImageList(ctx, options)
}

func (m MockClient) ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error) {
	return m.FnImagePull(ctx, refStr, options)
}

func (m MockClient) ServerVersion(ctx context.Context) (types.Version, error) {
	return m.FnServerVersion(ctx)
}

func (m MockClient) VolumeInspect(ctx context.Context, volumeID string) (volume.Volume, error) {
	return m.FnVolumeInspect(ctx, volumeID)
}

func (m MockClient) Info(ctx context.Context) (system.Info, error) {
	return m.FnInfo(ctx)
}
