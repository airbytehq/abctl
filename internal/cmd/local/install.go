package local

import (
	"github.com/airbytehq/abctl/internal/k8s"
)

type InstallOpts struct {
	HelmChartVersion  string
	HelmValuesYaml    string
	AirbyteChartLoc   string
	Secrets           []string
	Hosts             []string
	ExtraVolumeMounts []k8s.ExtraVolumeMount
	LocalStorage      bool
	EnablePsql17      bool

	DockerServer string
	DockerUser   string
	DockerPass   string
	DockerEmail  string

	NoBrowser bool
}

func (i *InstallOpts) DockerAuth() bool {
	return i.DockerUser != "" && i.DockerPass != ""
}

// TODO: Move the Install method and related functions from local/local/install.go to here
