package k8s

import (
	"context"
	"fmt"

	"github.com/airbytehq/abctl/internal/abctl"
)

// AbctlConfigFromData extracts abctl configuration from ConfigMap data
func AbctlConfigFromData(data map[string]string) (*abctl.Config, error) {
	config := &abctl.Config{}
	
	// Extract required values
	config.AirbyteAPIHost = data["AIRBYTE_API_HOST"]
	if config.AirbyteAPIHost == "" {
		return nil, fmt.Errorf("required field AIRBYTE_API_HOST not found or empty")
	}
	
	config.AirbyteURL = data["AIRBYTE_URL"]
	if config.AirbyteURL == "" {
		return nil, fmt.Errorf("required field AIRBYTE_URL not found or empty")
	}
	
	// Extract auth endpoint if available
	config.AirbyteAuthURL = data["AB_AIRBYTE_AUTH_IDENTITY_PROVIDER_OIDC_ENDPOINTS_AUTHORIZATION_SERVER_ENDPOINT"]
	
	return config, nil
}

// GetAbctlConfig gets the abctl configuration from k8s
func GetAbctlConfig(ctx context.Context, client Client, namespace string) (*abctl.Config, error) {
	configMap, err := client.ConfigMapGet(ctx, namespace, "abctl")
	if err != nil {
		return nil, fmt.Errorf("failed to load abctl config: %w (hint: run 'abctl init' first)", err)
	}

	apiHost := configMap.Data["airbyteApiHost"]
	if apiHost == "" {
		return nil, fmt.Errorf("airbyteApiHost not found in abctl config")
	}

	airbyteURL := configMap.Data["airbyteURL"]
	if airbyteURL == "" {
		return nil, fmt.Errorf("airbyteURL not found in abctl config")
	}

	authURL := configMap.Data["airbyteAuthURL"]

	return &abctl.Config{
		AirbyteAPIHost: apiHost,
		AirbyteURL:     airbyteURL,
		AirbyteAuthURL: authURL,
	}, nil
}