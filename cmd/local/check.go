package local

import (
	"context"
	"fmt"
	"github.com/airbytehq/abctl/internal/local/docker"
	"github.com/airbytehq/abctl/internal/local/localerr"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
	"net"
	"net/http"
	"strings"
	"time"
)

// dockerClient is exposed here primarily for testing purposes.
// A test should override this value to mock out a docker-client.
// If this value is nil, the default docker-client (as returned from defaultDocker) will be utilized.
var dockerClient *docker.Docker

// dockerInstalled checks if docker is installed on the host machine.
// Returns a nil error if docker was successfully detected, otherwise an error will be returned.  Any error returned
// is guaranteed to include the ErrDocker error in the error chain.
func dockerInstalled(ctx context.Context, t telemetry.Client) error {
	spinner, _ := pterm.DefaultSpinner.Start("docker - checking for docker installation")

	var err error
	if dockerClient == nil {
		if dockerClient, err = docker.New(); err != nil {
			spinner.Fail("docker - could not create client")
			return fmt.Errorf("%w: could not create client: %w", localerr.ErrDocker, err)
		}
	}

	v, err := dockerClient.Version(ctx)
	if err != nil {
		spinner.Fail("docker - could not communicate with the docker agent")
		return fmt.Errorf("%w: %w", localerr.ErrDocker, err)
	}

	t.Attr("docker_version", v.Version)
	t.Attr("docker_arch", v.Arch)
	t.Attr("docker_platform", v.Platform)

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

	if port < 1024 {
		spinner.Warning(fmt.Sprintf(
			"port %d - availability cannot be determined as this is a privileged port\n"+
				"(less than 1024), installation may not complete successfully",
			port))
		return nil
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
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

		if res.StatusCode == http.StatusUnauthorized && strings.Contains(res.Header.Get("WWW-Authenticate"), "abctl") {
			spinner.Success(fmt.Sprintf("port %d - port appears to be running a previous Airbyte installation", port))
			return nil
		}
	}
	// if we're able to bind to the port (and then release it), it should be available
	defer func() {
		_ = listener.Close()
	}()

	spinner.Success(fmt.Sprintf("port %d - appears to be available", port))
	return nil
}
