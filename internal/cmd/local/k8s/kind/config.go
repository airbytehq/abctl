package kind

import "github.com/airbytehq/abctl/internal/cmd/local/paths"

type Config struct {
	Kind       string `yaml:"kind"`
	ApiVersion string `yaml:"apiVersion"`
	Nodes      []Node `yaml:"nodes"`
}

type Node struct {
	Role   NodeRole          `yaml:"role"`
	Image  string            `yaml:"image"`
	Labels map[string]string `yaml:"labels"`

	ExtraMounts       []Mount       `yaml:"extraMounts"`
	ExtraPortMappings []PortMapping `yaml:"extraPortMappings"`

	// KubeadmConfigPatches are applied to the generated kubeadm config as
	// strategic merge patches to `kustomize build` internally
	// https://github.com/kubernetes/community/blob/a9cf5c8f3380bb52ebe57b1e2dbdec136d8dd484/contributors/devel/sig-api-machinery/strategic-merge-patch.md
	// This should be an inline yaml blob-string
	KubeadmConfigPatches []string `yaml:"kubeadmConfigPatches"`
}

type Mount struct {
	ContainerPath  string           `yaml:"containerPath"`
	HostPath       string           `yaml:"hostPath"`
	ReadOnly       bool             `yaml:"readOnly"`
	SelinuxRelabel bool             `yaml:"selinuxRelabel"`
	Propagation    MountPropagation `yaml:"propagation"`
}

type MountPropagation string

type NodeRole string

type PortMapping struct {
	ContainerPort int32               `yaml:"containerPort"`
	HostPort      int32               `yaml:"hostPort"`
	ListenAddress string              `yaml:"listenAddress"`
	Protocol      PortMappingProtocol `yaml:"protocol"`
}

type PortMappingProtocol string

func DefaultConfig() *Config {
	kubeadmConfigPatch := `kind: InitConfiguration
nodeRegistration:
  kubeletExtraArgs:
    node-labels: "ingress-ready=true"`

	cfg := &Config{
		Kind:       "Cluster",
		ApiVersion: "kind.x-k8s.io/v1alpha4",
		Nodes: []Node{
			{
				Role:                 "control-plane",
				KubeadmConfigPatches: []string{kubeadmConfigPatch},
				ExtraMounts: []Mount{
					{
						HostPath:      paths.Data,
						ContainerPath: "/var/local-path-provider",
					},
				},
				ExtraPortMappings: []PortMapping{
					{
						ContainerPort: 80,
						HostPort:      8000,
					},
				},
			},
		},
	}

	return cfg
}

func (c *Config) WithVolumeMount(hostPath string, containerPath string) *Config {
	c.Nodes[0].ExtraMounts = append(c.Nodes[0].ExtraMounts, Mount{HostPath: hostPath, ContainerPath: containerPath})
	return c
}

func (c *Config) WithHostPort(port int) *Config {
	c.Nodes[0].ExtraPortMappings[0].HostPort = int32(port)
	return c
}
