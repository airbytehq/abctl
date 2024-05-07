package local

import (
	"context"
	"fmt"
	"github.com/airbytehq/abctl/internal/local/docker"
	"github.com/airbytehq/abctl/internal/local/localerr"
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
func dockerInstalled(ctx context.Context) (docker.Version, error) {
	var err error
	if dockerClient == nil {
		if dockerClient, err = docker.New(ctx); err != nil {
			pterm.Error.Println("Could not create Docker client")
			return docker.Version{}, fmt.Errorf("%w: could not create client: %w", localerr.ErrDocker, err)
		}
	}

	version, err := dockerClient.Version(ctx)
	if err != nil {
		pterm.Error.Println("Could not communicate with the Docker daemon")
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
// bound to that port. If something behinds Airbyte is using it, then treat this as a inaccessible port.
func portAvailable(ctx context.Context, port int) error {
	if port < 1024 {
		pterm.Warning.Printfln(
			"Availability of port %d cannot be determined, as this is a privileged port (less than 1024).\n"+
				"Installation may not complete successfully",
			port)
		return nil
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		// check if an existing airbyte installation is already listening on this port
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://localhost:%d", port), nil)
		if err != nil {
			pterm.Error.Printfln("Port %d request could not be created", port)
			return fmt.Errorf("%w: could not create request: %w", localerr.ErrPort, err)
		}

		res, err := httpClient.Do(req)
		if err != nil {
			pterm.Error.Printfln("Port %d appears to already be in use", port)
			return fmt.Errorf("%w: could not send request: %w", localerr.ErrPort, err)
		}

		if res.StatusCode == http.StatusUnauthorized && strings.Contains(res.Header.Get("WWW-Authenticate"), "abctl") {
			pterm.Success.Printfln("Port %d appears to be running a previous Airbyte installation", port)
			return nil
		}
	}
	// if we're able to bind to the port (and then release it), it should be available
	defer func() {
		_ = listener.Close()
	}()

	pterm.Success.Printfln("Port %d appears to be available", port)
	return nil
}
