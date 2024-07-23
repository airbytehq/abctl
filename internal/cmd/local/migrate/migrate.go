package migrate

import (
	"context"
	"errors"
	"fmt"
	"github.com/airbytehq/abctl/internal/cmd/local/docker"
	"github.com/airbytehq/abctl/internal/cmd/local/paths"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/archive"
	"github.com/pterm/pterm"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	migratePGDATA = "/var/lib/postgresql/data"
	imgAlpine     = "alpine:3.20"
	imgPostgres   = "postgres:13-alpine"
)

// FromDockerVolume handles migrating the existing docker compose database into the abctl managed k8s cluster.
func FromDockerVolume(ctx context.Context, dockerCli docker.Client, volume string) error {
	if v := volumeExists(ctx, dockerCli, volume); v == "" {
		return errors.New(fmt.Sprintf("volume %s does not exist", volume))
	}

	if err := ensureImage(ctx, dockerCli, imgAlpine); err != nil {
		return err
	}
	if err := ensureImage(ctx, dockerCli, imgPostgres); err != nil {
		return err
	}

	// create a container for running the `docker cp` command
	// docker run -d -v airbyte_db:/var/lib/postgresql/data alpine:3.20 tail -f /dev/null
	conCopy, err := dockerCli.ContainerCreate(
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
		return fmt.Errorf("unable to create initial docker migration container: %w", err)
	}
	pterm.Debug.Println(fmt.Sprintf("Created initial migration container '%s'", conCopy.ID))

	// docker cp [conCopy.ID]]:/$migratePGDATA/. ~/.airbyte/abctl/data/airbyte-volume-db/pgdata
	dst := filepath.Join(paths.Data, "airbyte-volume-db", "pgdata")
	// ensure dst directory exists
	if err := os.MkdirAll(dst, 0766); err != nil {
		return fmt.Errorf("unable to create directory '%s': %w", dst, err)
	}
	// ensure the permissions are correct
	if err := os.Chmod(dst, 0777); err != nil {
		return fmt.Errorf("unable to chmod directory '%s': %w", dst, err)
	}

	// note the src must end with a `.`, due to how docker cp works with directories
	if err := copyFromContainer(ctx, dockerCli, conCopy.ID, migratePGDATA+"/.", dst); err != nil {
		return fmt.Errorf("unable to copy airbyte db data from container %s: %w", conCopy.ID, err)
	}
	pterm.Debug.Println(fmt.Sprintf("Copied airbyte db data from container '%s' to '%s'", conCopy.ID, dst))

	stopAndRemoveContainer(ctx, dockerCli, conCopy.ID)

	// Create a container for adding the correct db user and renaming the database.
	// We have inconsistencies between our docker and helm default database credentials and even our database name.
	// docker run
	// -e POSTGRES_USER=docker -e POSTGRES_PASSWORD=docker -e POSTGRES_DB=airbyte -e PGDATA=/var/lib/postgresql/data \
	// -v ~/.airbyte/abctl/data/airbyte-volume-db/pgdata:/var/lib/postgresql/data
	// postgres:13-alpine
	conTransform, err := dockerCli.ContainerCreate(
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
		return fmt.Errorf("unable to create docker container: %w", err)
	}
	pterm.Debug.Println(fmt.Sprintf("Created secondary migration container '%s'", conTransform.ID))
	pterm.Debug.Println(fmt.Sprintf("Container was created with the following warnings: %s", conTransform.Warnings))
	if err := dockerCli.ContainerStart(ctx, conTransform.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("unable to start container %s: %w", conTransform.ID, err)
	}
	// cleanup and remove container when we're done
	defer func() { stopAndRemoveContainer(ctx, dockerCli, conTransform.ID) }()

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
	if err := exec(ctx, dockerCli, conTransform.ID, cmdPsqlUser); err != nil {
		pterm.Debug.Println("Failed to add postgres user")
		return fmt.Errorf("unable to update postgres user: %w", err)
	}
	pterm.Debug.Println(fmt.Sprintf("Adding Airbyte postgres user completed in %s", time.Since(now)))

	// rename the database to match the default helm database name
	pterm.Debug.Println("Renaming database")
	now = time.Now()
	if err := exec(ctx, dockerCli, conTransform.ID, cmdPsqlRename); err != nil {
		pterm.Debug.Println("Failed to rename database")
		return fmt.Errorf("unable to rename postgres database: %w", err)
	}
	pterm.Debug.Println(fmt.Sprintf("Renaming database completed in %s", time.Since(now)))

	return nil
}

// volumeExists returns the MountPoint of the volumeID (if the volume exists), an empty string otherwise.
func volumeExists(ctx context.Context, d docker.Client, volumeID string) string {
	if v, err := d.VolumeInspect(ctx, volumeID); err != nil {
		pterm.Debug.Println(fmt.Sprintf("Volume %s cannot be accessed: %s", volumeID, err))
		return ""
	} else {
		return v.Mountpoint
	}
}

func ensureImage(ctx context.Context, d docker.Client, img string) error {
	// check if an image already exists on the host
	pterm.Debug.Println(fmt.Sprintf("Checking if the image '%s' already exists", img))
	filter := filters.NewArgs()
	filter.Add("reference", img)
	imgs, err := d.ImageList(ctx, image.ListOptions{Filters: filter})
	if err != nil {
		pterm.Error.Println(fmt.Sprintf("unable to list docker images when checking for '%s'", img))
		return fmt.Errorf("unable to list image '%s': %w", img, err)
	}

	// if it does exist, there is nothing else to do
	if len(imgs) > 0 {
		pterm.Debug.Println(fmt.Sprintf("Image '%s' already exists", img))
		return nil
	}

	pterm.Debug.Println(fmt.Sprintf("Image '%s' not found, pulling it", img))
	// if we're here, then we need to pull the image
	reader, err := d.ImagePull(ctx, img, image.PullOptions{})
	if err != nil {
		pterm.Error.Println(fmt.Sprintf("unable to pull the docker image '%s'", img))
		return fmt.Errorf("unable to pull image '%s': %w", img, err)
	}
	pterm.Debug.Println(fmt.Sprintf("Successfully pulled the docker image '%s'", img))
	defer reader.Close()
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return fmt.Errorf("error fetching output: %w", err)
	}

	return nil
}

// copyFromContainer emulates the `docker cp` command.
// The dst will be treated as a directory.
func copyFromContainer(ctx context.Context, d docker.Client, container, src, dst string) error {
	reader, stat, err := d.CopyFromContainer(ctx, container, src)
	if err != nil {
		return fmt.Errorf("unable to copy from container '%s': %w", container, err)
	}
	defer reader.Close()

	copyInfo := archive.CopyInfo{
		Path:   src,
		Exists: true,
		IsDir:  stat.Mode.IsDir(),
	}

	if err := archive.CopyTo(reader, copyInfo, dst); err != nil {
		return fmt.Errorf("unable to copy from container '%s': %w", container, err)
	}

	return nil
}

// stopAndRemoveContainer will stop and ultimately remove the containerID
func stopAndRemoveContainer(ctx context.Context, d docker.Client, containerID string) {
	pterm.Debug.Println(fmt.Sprintf("Stopping container '%s'", containerID))
	if err := d.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
		pterm.Debug.Println(fmt.Sprintf("Unable to stop docker container %s: %s", containerID, err))
	}
	pterm.Debug.Println(fmt.Sprintf("Removing container '%s'", containerID))
	if err := d.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		pterm.Debug.Println(fmt.Sprintf("Unable to remove docker container %s: %s", containerID, err))
	}
}

// Exec executes an exec cmd against the container.
// Largely inspired by the official docker client - https://github.com/docker/cli/blob/d69d501f699efb0cc1f16274e368e09ef8927840/cli/command/container/exec.go#L93
func exec(ctx context.Context, d docker.Client, container string, cmd []string) error {
	if _, err := d.ContainerInspect(ctx, container); err != nil {
		return fmt.Errorf("unable to inspect container '%s': %w", container, err)
	}

	resCreate, err := d.ContainerExecCreate(ctx, container, types.ExecConfig{Cmd: cmd})
	if err != nil {
		return fmt.Errorf("unable to create exec for container '%s': %w", container, err)
	}

	if err := d.ContainerExecStart(ctx, resCreate.ID, types.ExecStartCheck{}); err != nil {
		return fmt.Errorf("unable to start exec for container '%s': %w", container, err)
	}

	ticker := time.NewTicker(500 * time.Millisecond) // how often to check
	timer := time.After(5 * time.Minute)             // how long to wait
	running := true

	// loop until the exec command returns a "Running == false" status, or until we've hit our timer
	for running {
		select {
		case <-ticker.C:
			res, err := d.ContainerExecInspect(ctx, resCreate.ID)
			if err != nil {
				return fmt.Errorf("unable to inspect container '%s': %w", container, err)
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
