package local

import (
	"context"
	"errors"
	"fmt"
	"github.com/airbytehq/abctl/internal/local/localerr"
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

// serverVersioner exists for testing purposes.
type serverVersioner interface {
	ServerVersion(context.Context) (types.Version, error)
}

// dockerClient is exposed here primarily for testing purposes.
// A test should override this value to mock out a docker-client.
// If this value is nil, the default docker-client (as returned from defaultDocker) will be utilized.
var dockerClient serverVersioner

// defaultDocker returns a docker-client (serverVersioner) that is platform and user specific.
// The userHome is required for osx users given how docker configures itself.
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
		return nil, fmt.Errorf("%w: could not create docker client: %w", localerr.ErrDocker, err)
	}

	return docker, nil
}

// dockerInstalled checks if docker is installed on the host machine.
// Returns a nil error if docker was successfully detected, otherwise an error will be returned.  Any error returned
// is guaranteed to include the ErrDocker error in the error chain.
func dockerInstalled(ctx context.Context, t telemetry.Client, userHome string) error {
	spinner, _ := pterm.DefaultSpinner.Start("docker - checking for docker installation")

	var err error
	if dockerClient == nil {
		if dockerClient, err = defaultDocker(userHome); err != nil {
			spinner.Fail("docker - could not create client")
			return fmt.Errorf("%w: could not create client: %w", localerr.ErrDocker, err)
		}
	}

	v, err := dockerClient.ServerVersion(ctx)
	if err != nil {
		spinner.Fail("docker - could not communicate with the docker agent")
		return fmt.Errorf("%w: %w", localerr.ErrDocker, err)
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

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 3*time.Second)
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
		return fmt.Errorf("%w: could not dial tcp address: %w", localerr.ErrPort, err)
	}
	defer func() {
		_ = conn.Close()
	}()

	// check if an existing airbyte installation is already listening on this port
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://localhost:%d", port), nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("port %d - could not create request", port))
		return fmt.Errorf("%w: could not create request: %w", localerr.ErrPort, err)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		spinner.Fail(fmt.Sprintf("port %d - port is already in use", port))
		return fmt.Errorf("%w: could not send request: %w", localerr.ErrPort, err)
	}

	if res.StatusCode == 401 && strings.Contains(res.Header.Get("WWW-Authenticate"), "abctl") {
		spinner.Success(fmt.Sprintf("port %d - port appears to be running a previous Airbyte installation", port))
		return nil
	}

	spinner.Fail(fmt.Sprintf("port %d - port is unavailable", port))
	return fmt.Errorf("%w: port %d unavailable", localerr.ErrPort, port)
}
