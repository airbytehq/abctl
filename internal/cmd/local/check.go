package local

import (
	"context"
	"fmt"
	"github.com/airbytehq/abctl/internal/cmd/local/docker"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/pterm/pterm"
	"io"
	"net"
	"net/http"
	"time"
)

// dockerClient is exposed here primarily for testing purposes.
// A test should override this value to mock out a docker-client.
// If this value is nil, the default docker-client (as returned from defaultDocker) will be utilized.
var dockerClient *docker.Docker

// dockerInstalled checks if docker is installed on the host machine.
// Returns a nil error if docker was successfully detected, otherwise an error will be returned.  Any error returned
// is guaranteed to include the ErrDocker error in the error chain.
func dockerInstalled(ctx context.Context) (docker.Version, error) {
	var err error
	if dockerClient == nil {
		if dockerClient, err = docker.New(ctx); err != nil {
			pterm.Error.Println("Unable to create Docker client")
			return docker.Version{}, fmt.Errorf("%w: unable to create client: %w", localerr.ErrDocker, err)
		}
	}

	version, err := dockerClient.Version(ctx)
	if err != nil {
		pterm.Error.Println("Unable to communicate with the Docker daemon")
		return docker.Version{}, fmt.Errorf("%w: %w", localerr.ErrDocker, err)
	}
	pterm.Success.Println(fmt.Sprintf("Found Docker installation: version %s", version.Version))
	return version, nil

}

// doer interface for testing purposes
type doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// httpClient can be overwritten for testing purposes
var httpClient doer = &http.Client{Timeout: 3 * time.Second}

// portAvailable returns a nil error if the port is available, or already is use by Airbyte, otherwise returns an error.
//
// This function works by attempting to establish a tcp listener on a port.
// If we can establish a tcp listener on the port, an additional check is made to see if Airbyte may already be
// bound to that port. If something besides Airbyte is using it, treat this as an inaccessible port.
func portAvailable(ctx context.Context, port int) error {
	if port < 1024 {
		pterm.Warning.Printfln(
			"Availability of port %d cannot be determined, as this is a privileged port (less than 1024).\n"+
				"Installation may not complete successfully",
			port)
		return nil
	}

	// net.Listen doesn't support providing a context
	lc := &net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		pterm.Debug.Println(fmt.Sprintf("Unable to listen on port '%d': %s", port, err))

		// check if an existing airbyte installation is already listening on this port
		req, errInner := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://localhost:%d/api/v1/instance_configuration", port), nil)
		if errInner != nil {
			pterm.Error.Printfln("Port %d request could not be created", port)
			return fmt.Errorf("%w: unable to create request: %w", localerr.ErrPort, err)
		}

		res, errInner := httpClient.Do(req)
		if errInner != nil {
			pterm.Error.Printfln("Port %d appears to already be in use", port)
			return fmt.Errorf("%w: unable to send request: %w", localerr.ErrPort, err)
		}

		if res.StatusCode == http.StatusOK {
			pterm.Success.Printfln("Port %d appears to be running a previous Airbyte installation", port)
			return nil
		}

		// if we're here, we haven't been able to determine why this port may or may not be available
		body, errInner := io.ReadAll(res.Body)
		if errInner != nil {
			pterm.Debug.Println(fmt.Sprintf("Unable to read response body: %s", errInner))
		}
		pterm.Debug.Println(fmt.Sprintf(
			"Unable to determine if port '%d' is in use:\n  StatusCode: %d\n  Body: %s",
			port, res.StatusCode, body,
		))

		pterm.Error.Println(fmt.Sprintf(
			"Unable to determine if port '%d' is available, consider specifying a different port",
			port,
		))
		return fmt.Errorf("unable to determine if port '%d' is available: %w", port, err)
	}
	// if we're able to bind to the port (and then release it), it should be available
	defer func() {
		_ = listener.Close()
	}()

	pterm.Success.Printfln("Port %d appears to be available", port)
	return nil
}
