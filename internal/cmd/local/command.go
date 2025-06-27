package local

import (
	"net/http"

	"github.com/airbytehq/abctl/internal/docker"
	"github.com/airbytehq/abctl/internal/helm"
	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// BrowserLauncher primarily for testing purposes.
type BrowserLauncher func(url string) error

// Command is the local command, responsible for installing, uninstalling, or other local actions.
type Command struct {
	provider k8s.Provider
	docker   *docker.Docker

	http     HTTPClient
	helm     helm.Client
	k8s      k8s.Client
	portHTTP int
	spinner  *pterm.SpinnerPrinter
	tel      telemetry.Client
	launcher BrowserLauncher
	userHome string
}

// TODO: Move the constructor and other Command methods from local/local/cmd.go to here
