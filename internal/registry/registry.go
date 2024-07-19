package registry

import (
	"context"
	"fmt"
	"github.com/airbytehq/abctl/internal/build"
	"github.com/airbytehq/abctl/internal/cmd/local/docker"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"strconv"
)

type Registry struct {
	d *docker.Docker
}

const (
	dockerImage = "registry:2.8"
	port        = 6598
	name        = "abctl-registry"
)

func (r *Registry) Start(ctx context.Context) error {
	// check if the registry is already running
	if c, err := r.d.Client.ContainerInspect(ctx, name); err == nil {
		fmt.Println("found labels", c.Config.Labels)
		// container already running
		return nil
	}

	if err := r.d.ImagePullIfMissing(ctx, dockerImage); err != nil {
		return fmt.Errorf("unable to pull image '%s': %w", dockerImage, err)
	}

	if err := r.createContainer(ctx); err != nil {
		return fmt.Errorf("unable to create container: %w", err)
	}

	return nil
}

func (r *Registry) createContainer(ctx context.Context) error {
	// docker run -d -p 6598:5000 --restart always --name abctl-registry registry:2.8
	con, err := r.d.Client.ContainerCreate(
		ctx,
		&container.Config{
			Image: dockerImage,
			Labels: map[string]string{
				"io.airbyte.abctl-version": build.Version,
			},
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				"5000/tcp": []nat.PortBinding{{
					HostIP:   "0.0.0.0",
					HostPort: strconv.Itoa(port),
				}},
			},
			RestartPolicy: container.RestartPolicy{
				Name: container.RestartPolicyAlways,
			},
		},
		nil,
		nil,
		name,
	)
	if err != nil {
		return fmt.Errorf("unable to create container for image '%s': %w", dockerImage, err)
	}

	if err := r.d.Client.ContainerStart(ctx, con.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("unable to start container '%s': %w", con, err)
	}

	return nil
}
