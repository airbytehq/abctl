package airbox

import "github.com/airbytehq/abctl/internal/auth"

// CredentialStoreAdapter adapts ConfigStore to auth.CredentialsStore
type CredentialStoreAdapter struct {
	cfg ConfigStore
}

// NewCredentialStoreAdapter creates a new credentials store adapter
func NewCredentialStoreAdapter(cfg ConfigStore) *CredentialStoreAdapter {
	return &CredentialStoreAdapter{cfg: cfg}
}

// Load implements auth.CredentialsStore
func (a *CredentialStoreAdapter) Load() (*auth.Credentials, error) {
	config, err := a.cfg.Load()
	if err != nil {
		return nil, err
	}
	return config.GetCredentials()
}

// Save implements auth.CredentialsStore
func (a *CredentialStoreAdapter) Save(creds *auth.Credentials) error {
	config, err := a.cfg.Load()
	if err != nil {
		return err
	}

	config.SetCredentials(creds)

	return a.cfg.Save(config)
}
