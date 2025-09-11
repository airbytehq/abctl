package airbox

import (
	"fmt"
	"strings"
)

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

// ValidateCompanyIdentifier validates a company identifier for cloud deployment.
// This validation matches Airbyte platform SSO requirements: only checks for non-empty
// after trimming whitespace. The platform accepts any characters including spaces and
// special characters, so we defer complex validation to the server.
func ValidateCompanyIdentifier(companyID string) error {
	if strings.TrimSpace(companyID) == "" {
		return fmt.Errorf("company identifier is required")
	}
	return nil
}

// ValidateDataplaneName validates a dataplane name follows Kubernetes DNS-1123 subdomain rules.
// These constraints are required because dataplane names become Kubernetes resource names:
// - Max 63 characters (DNS label limit)
// - Start with lowercase letter (K8s requirement)
// - Only lowercase letters, numbers, hyphens (DNS-safe characters)
// - Cannot end with hyphen (DNS requirement)
func ValidateDataplaneName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if len(name) > 63 {
		return fmt.Errorf("name cannot exceed 63 characters")
	}

	// Must start with a letter
	if name[0] < 'a' || name[0] > 'z' {
		return fmt.Errorf("name must start with a lowercase letter")
	}

	// Only lowercase alphanumeric and hyphens allowed
	for i, char := range name {
		if (char < 'a' || char > 'z') &&
			(char < '0' || char > '9') &&
			char != '-' {
			return fmt.Errorf("name can only contain lowercase letters, numbers, and hyphens (invalid character at position %d)", i+1)
		}
	}

	// Cannot end with hyphen
	if name[len(name)-1] == '-' {
		return fmt.Errorf("name cannot end with a hyphen")
	}

	return nil
}
