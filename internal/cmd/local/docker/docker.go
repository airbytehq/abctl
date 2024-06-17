package docker

import (
	"context"
	"errors"
	"fmt"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pterm/pterm"
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
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error)
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ContainerRemove(ctx context.Context, container string, options container.RemoveOptions) error
	ContainerStop(ctx context.Context, container string, options container.StopOptions) error

	ContainerExecCreate(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error)
	ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error)
	ContainerExecStart(ctx context.Context, execID string, config types.ExecStartCheck) error

	ServerVersion(ctx context.Context) (types.Version, error)
	VolumeInspect(ctx context.Context, volumeID string) (volume.Volume, error)
}

var _ Client = (*client.Client)(nil)

// Docker for handling communication with the docker processes.
// Can be created with default settings by calling New or with a custom Client by manually instantiating this type.
type Docker struct {
	Client Client
}

// New returns a new Docker type with a default Client implementation.
func New(ctx context.Context) (*Docker, error) {
	// convert the client.NewClientWithOpts to a newPing function
	f := func(opts ...client.Opt) (pinger, error) {
		var p pinger
		var err error
		p, err = client.NewClientWithOpts(opts...)
		if err != nil {
			return nil, err
		}
		return p, nil
	}

	return newWithOptions(ctx, f, runtime.GOOS)
}

// newPing exists for testing purposes.
// This allows a mock docker client (client.Client) to be injected for tests
type newPing func(...client.Opt) (pinger, error)

// pinger interface for testing purposes.
// Adds the Ping method to the Client interface which is used by the New function.
type pinger interface {
	Client
	Ping(ctx context.Context) (types.Ping, error)
}

var _ pinger = (*client.Client)(nil)

// newWithOptions allows for the docker client to be injected for testing purposes.
func newWithOptions(ctx context.Context, newPing newPing, goos string) (*Docker, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine user home directory: %w", err)
	}

	var dockerCli Client
	dockerOpts := []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}

	switch goos {
	case "darwin":
		// on mac, sometimes the docker host isn't set correctly, if it fails check the home directory
		dockerCli, err = createAndPing(ctx, newPing, "unix:///var/run/docker.sock", dockerOpts)
		if err != nil {
			var err2 error
			dockerCli, err2 = createAndPing(ctx, newPing, fmt.Sprintf("unix://%s/.docker/run/docker.sock", userHome), dockerOpts)
			if err2 != nil {
				return nil, fmt.Errorf("%w: could not create docker client: (%w, %w)", localerr.ErrDocker, err, err2)
			}
			// if we made it here, clear out the original error,
			// as we were able to successfully connect on the second attempt
			err = nil
		}
	case "windows":
		dockerCli, err = createAndPing(ctx, newPing, "npipe:////./pipe/docker_engine", dockerOpts)
	default:
		dockerCli, err = createAndPing(ctx, newPing, "unix:///var/run/docker.sock", dockerOpts)
	}

	if err != nil {
		return nil, fmt.Errorf("%w: could not create docker client: %w", localerr.ErrDocker, err)
	}

	return &Docker{Client: dockerCli}, nil
}

// createAndPing attempts to create a docker client and ping it to ensure we can communicate
func createAndPing(ctx context.Context, newPing newPing, host string, opts []client.Opt) (Client, error) {
	cli, err := newPing(append(opts, client.WithHost(host))...)
	if err != nil {
		return nil, fmt.Errorf("could not create docker client: %w", err)
	}

	if _, err := cli.Ping(ctx); err != nil {
		return nil, fmt.Errorf("could not ping docker client: %w", err)
	}

	return cli, nil
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

func (d *Docker) VolumeExists(ctx context.Context, volumeID string) string {
	if v, err := d.Client.VolumeInspect(ctx, volumeID); err != nil {
		pterm.Debug.Println(fmt.Sprintf("Volume %s cannot be accessed: %s", volumeID, err))
		return ""
	} else {
		return v.Mountpoint
	}
}

const (
	migrateImage         = "postgresql:13-alpine"
	migrateContainerName = "airbyte-abctl-migrate"
	migrateUser          = "docker"
	migratePass          = "docker"
	migrateDB            = "airbyte"
	migratePGDATA        = "/var/lib/postgresql/data"
)

func (d *Docker) Migrate(ctx context.Context, volume string) error {
	if v := d.VolumeExists(ctx, volume); v == "" {
		return errors.New(fmt.Sprintf("volume %s does not exist", volume))
	}

	// docker run --name airbyte-abctl-migrate \
	// -e POSTGRES_USER=docker -e POSTGRES_PASSWORD=docker -e POSTGRES_DB=airbyte -e PGDATA=/var/lib/postgresql/data \
	// -v airbyte_db:/var/lib/postgresql/data
	// postgres:13-alpine
	con, err := d.Client.ContainerCreate(
		ctx,
		&container.Config{
			Env: []string{
				"POSTGRES_USER=" + migrateUser,
				"POSTGRES_PASSWORD=" + migratePass,
				"POSTGRES_DB=" + migrateDB,
				"PGDATA=" + migratePGDATA,
			},
			Image:   migrateImage,
			Volumes: map[string]struct{}{"airbyte_db:/var/lib/postgresql/data": {}},
		},
		nil,
		nil,
		nil,
		migrateContainerName,
	)
	if err != nil {
		return fmt.Errorf("could not create docker container: %w", err)
	}
	// cleanup the container when we're done
	defer func() {
		if err := d.Client.ContainerStop(ctx, con.ID, container.StopOptions{}); err != nil {
			pterm.Debug.Println(fmt.Sprintf("Could not stop docker container %s: %s", con.ID, err))
		}
		if err := d.Client.ContainerRemove(ctx, con.ID, container.RemoveOptions{Force: true}); err != nil {
			pterm.Debug.Println(fmt.Sprintf("Could not remove docker container %s: %s", con.ID, err))
		}
	}()

	// docker exec airbyte-abctl-migrate psql -U docker -c "CREATE ROLE airbyte SUPERUSER CREATEROLE CREATEDB REPLICATION BYPASSRLS LOGIN PASSWORD 'airbyte'"
	// docker exec airbyte-abctl-migrate psql -U airbyte postgres -c 'ALTER DATABASE airbyte RENAME TO "db-airbyte"'
	var (
		cmdPsqlUser   = `psql -U docker -c "CREATE ROLE airbyte SUPERUSER CREATEROLE CREATEDB REPLICATION BYPASSRLS LOGIN PASSWORD 'airbyte'"`
		cmdPsqlRename = `psql -U airbyte postgres -c 'ALTER DATABASE airbyte RENAME TO "db-airbyte"'`
	)
	// add a new database user to match the default helm user
	if err := d.Exec(ctx, con.ID, []string{cmdPsqlUser}); err != nil {
		return fmt.Errorf("could not update postgres user: %w", err)
	}
	// rename the database to match the default helm database name
	if err := d.Exec(ctx, con.ID, []string{cmdPsqlRename}); err != nil {
		return fmt.Errorf("could not rename postgres database: %w", err)
	}

	return nil
}

// Exec executes an exec cmd against the container.
// Largely inspired by the official docker client - https://github.com/docker/cli/blob/d69d501f699efb0cc1f16274e368e09ef8927840/cli/command/container/exec.go#L93
func (d *Docker) Exec(ctx context.Context, container string, cmd []string) error {
	if _, err := d.Client.ContainerInspect(ctx, container); err != nil {
		return fmt.Errorf("could not inspect container '%s': %w", container, err)
	}

	resCreate, err := d.Client.ContainerExecCreate(ctx, container, types.ExecConfig{Cmd: cmd})
	if err != nil {
		return fmt.Errorf("could not create exec for container '%s': %w", container, err)
	}

	if err := d.Client.ContainerExecStart(ctx, resCreate.ID, types.ExecStartCheck{}); err != nil {
		return fmt.Errorf("could not start exec for container '%s': %w", container, err)
	}

	resInspect, err := d.Client.ContainerExecInspect(ctx, resCreate.ID)
	if err != nil {
		return fmt.Errorf("could not inspect container '%s': %w", container, err)
	}

	if resInspect.ExitCode != 0 {
		return fmt.Errorf("container '%s' exec exited with non-zero exit code: %d", container, resInspect.ExitCode)
	}

	return nil
}
