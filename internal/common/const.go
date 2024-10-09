package common

const (
	AirbyteBootloaderPodName = "airbyte-abctl-airbyte-bootloader"
	AirbyteChartName         = "airbyte/airbyte"
	AirbyteChartRelease      = "airbyte-abctl"
	AirbyteIngress           = "ingress-abctl"
	AirbyteNamespace         = "airbyte-abctl"
	AirbyteRepoName          = "airbyte"
	AirbyteRepoURL           = "https://airbytehq.github.io/helm-charts"
	NginxChartName           = "nginx/ingress-nginx"
	NginxChartRelease        = "ingress-nginx"
	NginxNamespace           = "ingress-nginx"
	NginxRepoName            = "nginx"
	NginxRepoURL             = "https://kubernetes.github.io/ingress-nginx"

	// DockerAuthSecretName is the name of the secret which holds the docker authentication information.
	DockerAuthSecretName = "docker-auth"
)
