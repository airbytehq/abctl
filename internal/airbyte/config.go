package airbyte

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/airbytehq/abctl/internal/abctl"
	httputil "github.com/airbytehq/abctl/internal/http"
)

const (
	EditionEnterprise    = "enterprise"
	EditionCloud         = "cloud"
	CloudAuthMethodEmail = "email"
	CloudAuthMethodSSO   = "sso"
)

// CloudAuthMethod contains the name of the auth method and user friendly description.
type CloudAuthMethod struct {
	Name        string
	Description string
}

type CloudAuthMethods []CloudAuthMethod

// InstanceConfiguration represents the response from the instance_configuration endpoint
type InstanceConfiguration struct {
	Edition string `json:"edition"`
	Auth    struct {
		Mode                   string `json:"mode"`
		AuthorizationServerUrl string `json:"authorizationServerUrl,omitempty"`
	} `json:"auth"`
}

func DefaultCloudAuthMethods() CloudAuthMethods {
	return CloudAuthMethods{
		{
			Name:        CloudAuthMethodEmail,
			Description: "Email/Password",
		},
		{
			Name:        CloudAuthMethodSSO,
			Description: "SSO (Single Sign-On)",
		},
	}
}

// Descriptions returns all the Cloud auth method descriptions.
func (m CloudAuthMethods) Descriptions() []string {
	result := make([]string, len(m))
	for i, method := range m {
		result[i] = method.Description
	}
	return result
}

// BuildEnterpriseConfig creates an abctl.Config for Enterprise (SME) deployment
func BuildEnterpriseConfig(endpoint string, instanceConfig *InstanceConfiguration) (*abctl.Config, error) {
	// Normalize endpoint
	baseURL := strings.TrimSuffix(endpoint, "/")

	// Use discovered auth URL or fallback to default
	authURL := instanceConfig.Auth.AuthorizationServerUrl
	if authURL == "" {
		authURL = baseURL + "/auth/realms/airbyte" // Default fallback
	}

	// Construct configuration for SME
	config := &abctl.Config{
		AirbyteAPIHost: baseURL + "/api/v1",
		AirbyteURL:     baseURL,
		AirbyteAuthURL: authURL,
		OIDCClientID:   "airbyte-webapp", // Default OIDC client ID
		Edition:        instanceConfig.Edition,
	}

	return config, nil
}

// BuildCloudConfig creates an abctl.Config for Cloud deployment
func BuildCloudConfig(realmIdentifier string, method *CloudAuthMethod) (*abctl.Config, error) {
	if realmIdentifier == "" {
		return nil, fmt.Errorf("realm identifier is required for cloud deployment")
	}

	// Allow override of cloud domains via environment variables for testing
	cloudDomain := os.Getenv("AIRBYTE_CLOUD_DOMAIN")
	if cloudDomain == "" {
		cloudDomain = "cloud.airbyte.com"
	}

	cloudAPIDomain := os.Getenv("AIRBYTE_CLOUD_API_DOMAIN")
	if cloudAPIDomain == "" {
		cloudAPIDomain = "api.airbyte.com"
	}

	// Construct configuration for Cloud
	// Use either company-specific realm (SSO) or _airbyte-cloud-users (email/password)
	authRealmURL := fmt.Sprintf("https://%s/auth/realms/%s", cloudDomain, realmIdentifier)
	config := &abctl.Config{
		AirbyteAPIHost: fmt.Sprintf("https://%s", cloudAPIDomain),
		AirbyteURL:     fmt.Sprintf("https://%s", cloudDomain),
		AirbyteAuthURL: authRealmURL,
		OIDCClientID:   "airbyte-webapp", // Default OIDC client ID
		Edition:        "cloud",
	}

	return config, nil
}

// ValidateEnterpriseEndpoint checks if the SME endpoint supports OIDC authentication
func ValidateEnterpriseEndpoint(ctx context.Context, endpoint string) (*InstanceConfiguration, error) {
	// Normalize endpoint
	baseURL := strings.TrimSuffix(endpoint, "/")

	// Call api/v1/instance_configuration to check auth.mode
	configURL := baseURL + "/api/v1/instance_configuration"

	req, err := http.NewRequestWithContext(ctx, "GET", configURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := httputil.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SME endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SME endpoint returned status %d, expected 200", resp.StatusCode)
	}

	var config InstanceConfiguration
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode instance configuration: %w", err)
	}

	if config.Auth.Mode != "oidc" {
		return nil, fmt.Errorf("SME endpoint must have auth.mode=oidc, found: %s", config.Auth.Mode)
	}

	return &config, nil
}

// ValidateEndpointURL performs basic validation on an endpoint URL
func ValidateEndpointURL(endpoint string) error {
	if endpoint == "" {
		return fmt.Errorf("endpoint URL is required")
	}
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		return fmt.Errorf("endpoint must start with http:// or https://")
	}
	return nil
}

// ValidateCompanyIdentifier validates a company identifier for cloud deployment
func ValidateCompanyIdentifier(companyID string) error {
	if companyID == "" {
		return fmt.Errorf("company identifier is required")
	}
	// Basic validation for company identifier format
	if len(companyID) < 2 {
		return fmt.Errorf("company identifier must be at least 2 characters")
	}
	// Check for valid characters (alphanumeric and hyphens)
	for _, char := range companyID {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return fmt.Errorf("company identifier can only contain letters, numbers, hyphens, and underscores")
		}
	}
	return nil
}
