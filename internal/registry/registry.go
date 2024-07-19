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
	name        = "airbyte-abctl-registry"
)

// docker run -d -p 6598:5000 --restart always --name abctl-registry registry:2.8
func (r *Registry) Start(ctx context.Context) error {
	// check if the registry is already running
	if c, err := r.d.Client.ContainerInspect(ctx, name); err == nil {
		fmt.Println("id", c.ID)
		fmt.Println("name", c.Name)
		fmt.Println("found labels", c.Config.Labels)
		fmt.Println("running", c.State.Running)

		if c.State.Running {
			// already running nothing to do
			return nil
		}

		return r.start(ctx, c.ID)
	}

	if err := r.d.ImagePullIfMissing(ctx, dockerImage); err != nil {
		return fmt.Errorf("unable to pull image '%s': %w", dockerImage, err)
	}

	if containerID, err := r.create(ctx); err != nil {
		return err
	} else {
		return r.start(ctx, containerID)
	}
}

func (r *Registry) create(ctx context.Context) (string, error) {
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
		return "", fmt.Errorf("unable to create container for image '%s': %w", dockerImage, err)
	}

	return con.ID, nil
}

func (r *Registry) start(ctx context.Context, containerID string) error {
	if err := r.d.Client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("unable to start container '%s': %w", containerID, err)
	}

	return nil
}
