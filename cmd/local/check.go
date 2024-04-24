package local

import (
	"context"
	"errors"
	"fmt"
	"github.com/airbytehq/abctl/internal/local"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pterm/pterm"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const (
	host = "localhost"
)

type serverVersioner interface {
	ServerVersion(context.Context) (types.Version, error)
}

var dockerClient serverVersioner = func() serverVersioner {
	return nil
}()

func defaultDocker(userHome string) (serverVersioner, error) {
	var docker serverVersioner
	var err error

	switch runtime.GOOS {
	case "darwin":
		// on mac, sometimes the docker host isn't set correctly, if it fails check the home directory
		docker, err = client.NewClientWithOpts(client.FromEnv, client.WithHost("unix:///var/run/docker.sock"))
		if err != nil {
			// keep the original error, as we'll join with the next error (if another error occurs)
			outerErr := err
			docker, err = client.NewClientWithOpts(client.FromEnv, client.WithHost(fmt.Sprintf("unix:///%s/.docker/run/docker.sock", userHome)))
			if err != nil {
				err = fmt.Errorf("%w: %w", err, outerErr)
			}
		}
	case "windows":
		docker, err = client.NewClientWithOpts(client.FromEnv, client.WithHost("npipe:////./pipe/docker_engine"))
	default:
		docker, err = client.NewClientWithOpts(client.FromEnv, client.WithHost("unix:///var/run/docker.sock"))
	}
	if err != nil {
		return nil, fmt.Errorf("%w: could not create docker client: %w", local.ErrDocker, err)
	}

	return docker, nil
}

func dockerInstalled(ctx context.Context, t telemetry.Client, userHome string) error {
	spinner, _ := pterm.DefaultSpinner.Start("docker - checking for docker installation")

	var err error
	if dockerClient == nil {
		if dockerClient, err = defaultDocker(userHome); err != nil {
			spinner.Fail("docker - could not create client")
			return fmt.Errorf("%w: could not create client: %w", local.ErrDocker, err)
		}
	}

	v, err := dockerClient.ServerVersion(ctx)
	if err != nil {
		spinner.Fail("docker - could not communicate with the docker agent")
		return fmt.Errorf("%w: %w", local.ErrDocker, err)
	}

	t.Attr("docker_version", v.Version)
	t.Attr("docker_arch", v.Arch)
	t.Attr("docker_platform", v.Platform.Name)

	spinner.Success(fmt.Sprintf("docker - found; version: %s", v.Version))
	return nil
}

// doer interface for testing purposes
type doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// httpClient can be overwritten for testing purposes
var httpClient doer = &http.Client{Timeout: 3 * time.Second}

// portAvailable returns a nil error if the port is available, or already is use by Airbyte, otherwise returns an error.
//
// This function works by attempting to establish a tcp connection to the port.
// If this connection fails with a "connection refused" message, the assumption is that the port ia actually available.
// If we can establish a tcp connection to the port, an additional check is made to see if Airbyte may already be
// bound to that port. If something behinds Airbyte is using it, then treat this as a inaccessible port.
func portAvailable(ctx context.Context, port int) error {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("port %d - checking port availability", port))

	server, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		spinner.Fail(fmt.Sprintf("port %d - could not resolve host tcp address", port))
		return fmt.Errorf("could not resolve host tcp address: %w", err)
	}

	conn, err := net.DialTCP("tcp", nil, server)
	if err != nil {
		var opErr *net.OpError
		if errors.As(err, &opErr) {
			// if the connection is refused, that should mean the port is actually available.
			if strings.Contains(opErr.Err.Error(), "connection refused") {
				spinner.Success(fmt.Sprintf("port %d - port is available", port))
				return nil
			}
		}

		spinner.Fail(fmt.Sprintf("port %d - could not dial tcp address", port))
		return fmt.Errorf("could not dial tcp address: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	// check if an existing airbyte installation is already listening on this port
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://localhost:%d", port), nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("port %d - could not create request", port))
		return fmt.Errorf("could not create request: %w", err)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		// check for connection reset by peer
		var opErr *net.OpError
		if errors.As(err, &opErr) {
			// if the connection fails due to a reset error, that appears to mean the port is actually available.
			if strings.Contains(opErr.Err.Error(), "connection reset by peer") {
				spinner.Success(fmt.Sprintf("port %d - port is available", port))
				return nil
			}
		}

		spinner.Fail(fmt.Sprintf("port %d - port is already in use", port))
		return fmt.Errorf("could not send request: %w", err)
	}

	if res.StatusCode == 401 && strings.Contains(res.Header.Get("WWW-Authenticate"), "abctl") {
		spinner.Success(fmt.Sprintf("port %d - port appears to be running a previous Airbyte installation", port))
		return nil
	}

	spinner.Fail(fmt.Sprintf("port %d - port is already in use", port))
	return fmt.Errorf("port %d already in use", port)
}
