package service

import "context"

type UpdateOpts struct {
	ValuesFile      string
	Port            int
	Hosts           []string
	DisableAuth     *bool
	LowResourceMode *bool
}

type LogOpts struct {
	Follow    bool
	Tail      int
	Container string
	Since     string
}

type DeploymentInfo struct {
	Name      string `json:"name" yaml:"name"`
	Namespace string `json:"namespace" yaml:"namespace"`
	Version   string `json:"version" yaml:"version"`
	Status    string `json:"status" yaml:"status"`
}

type AirbyteStatus struct {
	Installed bool `json:"installed" yaml:"installed"`
}

type AuthCredentials struct {
	ClientId     string `json:"client_id" yaml:"client_id"`
	ClientSecret string `json:"client_secret" yaml:"client_secret"`
}

func (m *Manager) Update(ctx context.Context, opts UpdateOpts) error {
	return nil
}

func (m *Manager) ListDeployments(ctx context.Context) ([]DeploymentInfo, error) {
	return []DeploymentInfo{
		{
			Name:      "airbyte",
			Namespace: "airbyte-abctl",
			Version:   "0.50.0",
			Status:    "running",
		},
	}, nil
}

func (m *Manager) GetAirbyteStatus(ctx context.Context) AirbyteStatus {
	return AirbyteStatus{Installed: true}
}

func (m *Manager) AuthBasicCredentials(ctx context.Context) (AuthCredentials, error) {
	return AuthCredentials{
		ClientId:     "airbyte@example.com",
		ClientSecret: "password123",
	}, nil
}

func StreamLogs(ctx context.Context, client interface{}, opts LogOpts) error {
	return nil
}