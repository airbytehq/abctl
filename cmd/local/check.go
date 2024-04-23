package local

import (
	"context"
	"errors"
	"fmt"
	"github.com/pterm/pterm"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	host = "localhost"
)

// doer interface for testing purposes
type doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// httpClient can be overwritten for testing purposes
var httpClient doer = &http.Client{Timeout: 3 * time.Second}

// portAvailable returns true if the port is available, false otherwise.
//
// This function works by attempting to establish a tcp connection to the port.
// If this connection fails with a "connection refused" message, the assumption is that the port ia actually available.
// If we can establish a tcp connection to the port, then the assumption is that the port is already is use.
func portAvailable(ctx context.Context, port int) (bool, error) {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("port %d - checking port availability", port))

	server, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		spinner.Fail(fmt.Sprintf("port %d - could not resolve host tcp address: %s", port, err.Error()))
		return false, nil
	}

	conn, err := net.DialTCP("tcp", nil, server)
	if err != nil {
		var opErr *net.OpError
		if errors.As(err, &opErr) {
			// if the connection is refused, that should mean the port is actually available.
			if strings.Contains(opErr.Err.Error(), "connection refused") {
				spinner.Success(fmt.Sprintf("port %d - port is available", port))
				return true, nil
			}
		}
		return false, nil
	}
	defer func() {
		_ = conn.Close()
	}()

	// check if an existing airbyte installation is already listening on this port
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://localhost:%d", port), nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("port %d - could not create request", port))
		return false, fmt.Errorf("could not create request: %w", err)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		// check for connection reset by peer
		var opErr *net.OpError
		if errors.As(err, &opErr) {
			// if the connection fails due to a reset error, that appears to mean the port is actually available.
			if strings.Contains(opErr.Err.Error(), "connection reset by peer") {
				spinner.Success(fmt.Sprintf("port %d - port is available", port))
				return true, nil
			}
		}

		spinner.Fail(fmt.Sprintf("port %d - port is already in use", port))
		return false, fmt.Errorf("could not send request: %w", err)
	}

	if res.StatusCode == 401 && strings.Contains(res.Header.Get("WWW-Authenticate"), "abctl") {
		spinner.Success(fmt.Sprintf("port %d - port appears to be running a previous Airbyte installation", port))
		return true, nil
	}

	spinner.Fail(fmt.Sprintf("port %d - port is already in use", port))
	return false, nil
}
