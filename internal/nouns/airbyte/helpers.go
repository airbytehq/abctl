package airbyte

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/airbytehq/abctl/internal/docker"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
)

var dockerAPI *docker.Docker

func dockerInstalled(ctx context.Context, telClient telemetry.Client) (bool, error) {
	telClient.Attr("docker_version", "")
	if dockerAPI == nil {
		var err error
		dockerAPI, err = docker.New(ctx)
		if err != nil {
			return false, err
		}
	}
	d, err := dockerAPI.Version(ctx)
	if err != nil {
		return false, err
	}
	if d.Version == "" {
		return false, errors.New("unable to determine docker version")
	}
	telClient.Attr("docker_version", d.Version)
	pterm.Debug.Println(fmt.Sprintf("Determined docker version to be: %s", d.Version))
	return true, nil
}

func getPort(ctx context.Context, name string) (int, error) {
	// Simplified port detection - return default port for now
	return 8000, nil
}

func portAvailable(ctx context.Context, port int) error {
	host := "127.0.0.1:" + strconv.Itoa(port)

	listener, err := net.Listen("tcp", host)
	if err != nil {
		return fmt.Errorf("port %d is not available", port)
	}
	_ = listener.Close()

	return nil
}

func validateHostFlag(host string) error {
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	parsed, err := url.Parse("http://" + host)
	if err != nil {
		return fmt.Errorf("unable to parse host '%s': %w", host, err)
	}

	if parsed.Hostname() == "" {
		return fmt.Errorf("host '%s' does not appear to be valid", host)
	}

	if strings.Contains(parsed.Hostname(), " ") {
		return fmt.Errorf("host '%s' should not contain spaces", host)
	}

	return nil
}

func checkAirbyteDir() error {
	const airbyteDir = "/tmp/airbyte"
	fileInfo, err := os.Stat(airbyteDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("unable to determine status of '%s': %w", airbyteDir, err)
	}

	if !fileInfo.IsDir() {
		return errors.New(airbyteDir + " is not a directory")
	}

	if fileInfo.Mode().Perm() >= 0o744 {
		return nil
	}

	if err := os.Chmod(airbyteDir, 0o744); err != nil {
		return fmt.Errorf("unable to change permissions of '%s': %w", airbyteDir, err)
	}

	return nil
}