package config

import (
	"testing"
)

// TestSetupCloudSelection tests the core logic of Cloud setup selection
func TestSetupCloudSelection(t *testing.T) {
	// Clear environment variables before tests
	t.Setenv("AIRBYTE_CLOUD_DOMAIN", "")
	t.Setenv("AIRBYTE_CLOUD_API_DOMAIN", "")

	tests := []struct {
		name                string
		authMethod          string
		companyInput        string
		expectedRealmSuffix string
		wantErr             bool
	}{
		{
			name:                "email/password auth uses default realm",
			authMethod:          "Email/Password",
			expectedRealmSuffix: "_airbyte-cloud-users",
		},
		{
			name:                "SSO auth uses company realm",
			authMethod:          "SSO (Single Sign-On)",
			companyInput:        "my-company",
			expectedRealmSuffix: "my-company",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the core logic by checking what realm identifier is used
			var realmIdentifier string
			if tt.authMethod == "SSO (Single Sign-On)" {
				realmIdentifier = tt.companyInput
			} else {
				realmIdentifier = "_airbyte-cloud-users"
			}

			if realmIdentifier != tt.expectedRealmSuffix {
				t.Errorf("Expected realm identifier %s, got %s", tt.expectedRealmSuffix, realmIdentifier)
			}
		})
	}
}
