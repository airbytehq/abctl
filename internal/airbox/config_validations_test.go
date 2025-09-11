package airbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateEndpointURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid https URL",
			input:   "https://example.com",
			wantErr: false,
		},
		{
			name:    "valid http URL",
			input:   "http://localhost:8000",
			wantErr: false,
		},
		{
			name:    "empty URL",
			input:   "",
			wantErr: true,
		},
		{
			name:    "missing scheme",
			input:   "example.com",
			wantErr: true,
		},
		{
			name:    "invalid scheme",
			input:   "ftp://example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEndpointURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEndpointURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCompanyIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid identifier",
			input:   "my-company",
			wantErr: false,
		},
		{
			name:    "valid identifier with numbers",
			input:   "company-123",
			wantErr: false,
		},
		{
			name:    "valid with spaces (now allowed)",
			input:   "my company",
			wantErr: false,
		},
		{
			name:    "valid with special characters (now allowed)",
			input:   "my@company",
			wantErr: false,
		},
		{
			name:    "empty identifier",
			input:   "",
			wantErr: true,
		},
		{
			name:    "only whitespace",
			input:   "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCompanyIdentifier(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCompanyIdentifier() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDataplaneName(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:  "valid name",
			input: "my-dataplane",
		},
		{
			name:  "valid name with numbers",
			input: "dataplane-123",
		},
		{
			name:  "valid single character",
			input: "a",
		},
		{
			name:          "empty name",
			input:         "",
			expectedError: "name cannot be empty",
		},
		{
			name:          "too long name",
			input:         "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl", // 64 characters
			expectedError: "name cannot exceed 63 characters",
		},
		{
			name:          "starts with uppercase",
			input:         "My-dataplane",
			expectedError: "name must start with a lowercase letter",
		},
		{
			name:          "starts with number",
			input:         "1dataplane",
			expectedError: "name must start with a lowercase letter",
		},
		{
			name:          "starts with hyphen",
			input:         "-dataplane",
			expectedError: "name must start with a lowercase letter",
		},
		{
			name:          "contains uppercase",
			input:         "my-Dataplane",
			expectedError: "name can only contain lowercase letters, numbers, and hyphens (invalid character at position 4)",
		},
		{
			name:          "contains special character",
			input:         "my_dataplane",
			expectedError: "name can only contain lowercase letters, numbers, and hyphens (invalid character at position 3)",
		},
		{
			name:          "contains space",
			input:         "my dataplane",
			expectedError: "name can only contain lowercase letters, numbers, and hyphens (invalid character at position 3)",
		},
		{
			name:          "ends with hyphen",
			input:         "my-dataplane-",
			expectedError: "name cannot end with a hyphen",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDataplaneName(tt.input)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
