package oidc

import (
	"fmt"
	"os"
	"path/filepath"
)

// CredentialsPath returns the path to the credentials file
func CredentialsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	
	abctlDir := filepath.Join(homeDir, ".abctl")
	return filepath.Join(abctlDir, "credentials"), nil
}

// SaveCredentials saves credentials to file
func SaveCredentials(creds *Credentials) error {
	path, err := CredentialsPath()
	if err != nil {
		return err
	}
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Serialize credentials
	data, err := creds.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize credentials: %w", err)
	}
	
	// Write to file with restricted permissions
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}
	
	return nil
}

// LoadCredentials loads credentials from file
func LoadCredentials() (*Credentials, error) {
	path, err := CredentialsPath()
	if err != nil {
		return nil, err
	}
	
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no credentials found, please login first")
		}
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}
	
	creds, err := FromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}
	
	return creds, nil
}

// DeleteCredentials removes the credentials file
func DeleteCredentials() error {
	path, err := CredentialsPath()
	if err != nil {
		return err
	}
	
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}
	
	return nil
}