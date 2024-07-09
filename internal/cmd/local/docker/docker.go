package docker

import (
	"context"
	"errors"
	"fmt"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/cmd/local/paths"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pterm/pterm"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
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
	ContainerStart(ctx context.Context, container string, options container.StartOptions) error
	ContainerStop(ctx context.Context, container string, options container.StopOptions) error
	CopyFromContainer(ctx context.Context, container, srcPath string) (io.ReadCloser, types.ContainerPathStat, error)

	ContainerExecCreate(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error)
	ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error)
	ContainerExecStart(ctx context.Context, execID string, config types.ExecStartCheck) error

	ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error)
	ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error)

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
	var (
		dockerCli Client
		err       error
	)

	dockerOpts := []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}

	switch goos {
	case "darwin":
		// on mac, sometimes the docker host isn't set correctly, if it fails check the home directory
		dockerCli, err = createAndPing(ctx, newPing, "unix:///var/run/docker.sock", dockerOpts)
		if err != nil {
			var err2 error
			dockerCli, err2 = createAndPing(ctx, newPing, fmt.Sprintf("unix://%s/.docker/run/docker.sock", paths.UserHome), dockerOpts)
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

const (
	migratePGDATA = "/var/lib/postgresql/data"
	imgAlpine     = "alpine:3.20"
	imgPostgres   = "postgres:13-alpine"
)

// MigrateComposeDB handles migrating the existing docker compose database into the abctl managed k8s cluster.
// TODO: move this method out of the the docker class?
func (d *Docker) MigrateComposeDB(ctx context.Context, volume string) error {
	if v := d.volumeExists(ctx, volume); v == "" {
		return errors.New(fmt.Sprintf("volume %s does not exist", volume))
	}

	if err := d.ensureImage(ctx, imgAlpine); err != nil {
		return err
	}
	if err := d.ensureImage(ctx, imgPostgres); err != nil {
		return err
	}

	// create a container for running the `docker cp` command
	// docker run -d -v airbyte_db:/var/lib/postgresql/data alpine:3.20 tail -f /dev/null
	conCopy, err := d.Client.ContainerCreate(
		ctx,
		&container.Config{
			Image:      imgAlpine,
			Entrypoint: []string{"tail", "-f", "/dev/null"},
		},
		&container.HostConfig{
			Mounts: []mount.Mount{{
				Type:   mount.TypeVolume,
				Source: volume,
				Target: migratePGDATA,
			}},
		},
		nil,
		nil,
		"")
	if err != nil {
		return fmt.Errorf("could not create initial docker migration container: %w", err)
	}
	pterm.Debug.Println(fmt.Sprintf("Created initial migration container '%s'", conCopy.ID))

	// docker cp [conCopy.ID]]:/$migratePGDATA/. ~/.airbyte/abctl/data/airbyte-volume-db/pgdata
	dst := filepath.Join(paths.Data, "airbyte-volume-db", "pgdata")
	// ensure dst directory exists
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("could not create directory '%s': %w", dst, err)
	}
	// ensure the permissions are correct
	if err := os.Chmod(dst, 0777); err != nil {
		return fmt.Errorf("could not chmod directory '%s': %w", dst, err)
	}

	// note the src must end with a `.`, due to how docker cp works with directories
	if err := d.copyFromContainer(ctx, conCopy.ID, migratePGDATA+"/.", dst); err != nil {
		return fmt.Errorf("could not copy airbyte db data from container %s: %w", conCopy.ID, err)
	}
	pterm.Debug.Println(fmt.Sprintf("Copied airbyte db data from container '%s' to '%s'", conCopy.ID, dst))

	d.stopAndRemoveContainer(ctx, conCopy.ID)

	// Create a container for adding the correct db user and renaming the database.
	// We have inconsistencies between our docker and helm default database credentials and even our database name.
	// docker run
	// -e POSTGRES_USER=docker -e POSTGRES_PASSWORD=docker -e POSTGRES_DB=airbyte -e PGDATA=/var/lib/postgresql/data \
	// -v ~/.airbyte/abctl/data/airbyte-volume-db/pgdata:/var/lib/postgresql/data
	// postgres:13-alpine
	conTransform, err := d.Client.ContainerCreate(
		ctx,
		&container.Config{
			Image: imgPostgres,
			Env: []string{
				"POSTGRES_USER=docker",
				"POSTGRES_PASSWORD=docker",
				"POSTGRES_DB=postgres",
				"PGDATA=" + migratePGDATA,
			},
		},
		&container.HostConfig{
			Mounts: []mount.Mount{{
				Type:   mount.TypeBind,
				Source: dst,
				Target: migratePGDATA,
			}},
		},
		nil,
		nil,
		"")
	if err != nil {
		return fmt.Errorf("could not create docker container: %w", err)
	}
	pterm.Debug.Println(fmt.Sprintf("Created secondary migration container '%s'", conTransform.ID))
	pterm.Debug.Println(fmt.Sprintf("Container was created with the following warnings: %s", conTransform.Warnings))
	if err := d.Client.ContainerStart(ctx, conTransform.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("could not start container %s: %w", conTransform.ID, err)
	}
	// cleanup and remove container when we're done
	defer func() { d.stopAndRemoveContainer(ctx, conTransform.ID) }()

	// TODO figure out a better way to determine when the container has successfully started
	time.Sleep(10 * time.Second)

	// docker exec airbyte-abctl-migrate psql -U docker -c "CREATE ROLE airbyte SUPERUSER CREATEROLE CREATEDB REPLICATION BYPASSRLS LOGIN PASSWORD 'airbyte'"
	// docker exec airbyte-abctl-migrate psql -U airbyte postgres -c 'ALTER DATABASE airbyte RENAME TO "db-airbyte"'
	var (
		cmdPsqlRename = []string{"psql", "-U", "docker", "-d", "postgres", "-c", `ALTER DATABASE "airbyte" RENAME TO "db-airbyte"`}
		cmdPsqlUser   = []string{"psql", "-U", "docker", "-d", "postgres", "-c", `CREATE ROLE airbyte SUPERUSER CREATEROLE CREATEDB REPLICATION BYPASSRLS LOGIN PASSWORD 'airbyte'`}
	)
	// add a new database user to match the default helm user
	now := time.Now()
	pterm.Debug.Println("Adding Airbyte postgres user")
	if err := d.exec(ctx, conTransform.ID, cmdPsqlUser); err != nil {
		pterm.Debug.Println("Failed to add postgres user")
		return fmt.Errorf("could not update postgres user: %w", err)
	}
	pterm.Debug.Println(fmt.Sprintf("Adding Airbyte postgres user completed in %s", time.Since(now)))

	// rename the database to match the default helm database name
	pterm.Debug.Println("Renaming database")
	now = time.Now()
	if err := d.exec(ctx, conTransform.ID, cmdPsqlRename); err != nil {
		pterm.Debug.Println("Failed to rename database")
		return fmt.Errorf("could not rename postgres database: %w", err)
	}
	pterm.Debug.Println(fmt.Sprintf("Renaming database completed in %s", time.Since(now)))

	return nil
}

func (d *Docker) ensureImage(ctx context.Context, img string) error {
	// check if an image already exists on the host
	pterm.Debug.Println(fmt.Sprintf("Checking if the image '%s' already exists", img))
	filter := filters.NewArgs()
	filter.Add("reference", img)
	imgs, err := d.Client.ImageList(ctx, image.ListOptions{Filters: filter})
	if err != nil {
		pterm.Error.Println(fmt.Sprintf("Could not list docker images when checking for '%s'", img))
		return fmt.Errorf("could not list image '%s': %w", img, err)
	}

	// if it does exist, there is nothing else to do
	if len(imgs) > 0 {
		pterm.Debug.Println(fmt.Sprintf("Image '%s' already exists", img))
		return nil
	}

	pterm.Debug.Println(fmt.Sprintf("Image '%s' not found, pulling it", img))
	// if we're here, then we need to pull the image
	reader, err := d.Client.ImagePull(ctx, img, image.PullOptions{})
	if err != nil {
		pterm.Error.Println(fmt.Sprintf("Could not pull the docker image '%s'", img))
		return fmt.Errorf("could not pull image '%s': %w", img, err)
	}
	pterm.Debug.Println(fmt.Sprintf("Successfully pulled the docker image '%s'", img))
	defer reader.Close()
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return fmt.Errorf("error fetching output: %w", err)
	}

	return nil
}

// volumeExists returns the MountPoint of the volumeID (if the volume exists), an empty string otherwise.
func (d *Docker) volumeExists(ctx context.Context, volumeID string) string {
	if v, err := d.Client.VolumeInspect(ctx, volumeID); err != nil {
		pterm.Debug.Println(fmt.Sprintf("Volume %s cannot be accessed: %s", volumeID, err))
		return ""
	} else {
		return v.Mountpoint
	}
}

// Exec executes an exec cmd against the container.
// Largely inspired by the official docker client - https://github.com/docker/cli/blob/d69d501f699efb0cc1f16274e368e09ef8927840/cli/command/container/exec.go#L93
func (d *Docker) exec(ctx context.Context, container string, cmd []string) error {
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

	ticker := time.NewTicker(500 * time.Millisecond) // how often to check
	timer := time.After(5 * time.Minute)             // how long to wait
	running := true

	// loop until the exec command returns a "Running == false" status, or until we've hit our timer
	for running {
		select {
		case <-ticker.C:
			res, err := d.Client.ContainerExecInspect(ctx, resCreate.ID)
			if err != nil {
				return fmt.Errorf("could not inspect container '%s': %w", container, err)
			}
			running = res.Running
			if res.ExitCode != 0 {
				return fmt.Errorf("container '%s' exec exited with non-zero exit code: %d", container, res.ExitCode)
			}
		case <-timer:
			return errors.New("timed out waiting for docker exec to complete")
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// copyFromContainer emulates the `docker cp` command.
// The dst will be treated as a directory.
func (d *Docker) copyFromContainer(ctx context.Context, container, src, dst string) error {
	reader, stat, err := d.Client.CopyFromContainer(ctx, container, src)
	if err != nil {
		return fmt.Errorf("could not copy from container '%s': %w", container, err)
	}
	defer reader.Close()

	copyInfo := archive.CopyInfo{
		Path:   src,
		Exists: true,
		IsDir:  stat.Mode.IsDir(),
	}

	if err := archive.CopyTo(reader, copyInfo, dst); err != nil {
		return fmt.Errorf("could not copy from container '%s': %w", container, err)
	}

	return nil
}

// stopAndRemoveContainer will stop and ultimately remove the containerID
func (d *Docker) stopAndRemoveContainer(ctx context.Context, containerID string) {
	pterm.Debug.Println(fmt.Sprintf("Stopping container '%s'", containerID))
	if err := d.Client.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
		pterm.Debug.Println(fmt.Sprintf("Could not stop docker container %s: %s", containerID, err))
	}
	pterm.Debug.Println(fmt.Sprintf("Removing container '%s'", containerID))
	if err := d.Client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		pterm.Debug.Println(fmt.Sprintf("Could not remove docker container %s: %s", containerID, err))
	}
}
