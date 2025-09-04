package config

import (
	"context"
	"fmt"

	"github.com/airbytehq/abctl/internal/abctl"
	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/airbyte"
	"github.com/airbytehq/abctl/internal/ui"
)

// InitCmd represents the init command
type InitCmd struct {
	Force bool `flag:"" help:"Overwrite existing airbox configuration."`
}

// Run executes the init command
func (c *InitCmd) Run(ctx context.Context, cfg airbox.ConfigProvider, ui ui.Provider) error {
	ui.Title("Initializing airbox configuration")

	// Use UI guided setup to create config
	config, err := c.uiGuidedSetup(ctx, ui)
	if err != nil {
		return err
	}

	ui.ShowSection("Configuration:",
		fmt.Sprintf("Airbyte API Host: %s", config.AirbyteAPIHost),
		fmt.Sprintf("Airbyte URL: %s", config.AirbyteURL),
		fmt.Sprintf("Airbyte Auth URL: %s", config.AirbyteAuthURL),
	)

	// Load or create local airbox config
	abCfg, err := cfg.Load()
	if err != nil {
		return fmt.Errorf("failed to load airbox config: %w", err)
	}

	// Check if config already exists and not forcing
	if len(abCfg.Contexts) > 0 && !c.Force {
		return fmt.Errorf("airbox configuration already exists, use --force to overwrite")
	}

	// Create default context from config
	contextName := "default"
	if config.Edition == "cloud" {
		contextName = "cloud"
	} else {
		contextName = "enterprise"
	}

	context := airbox.Context{
		AirbyteAPIHost: config.AirbyteAPIHost,
		AirbyteURL:     config.AirbyteURL,
		AirbyteAuthURL: config.AirbyteAuthURL,
		OIDCClientID:   config.OIDCClientID,
		Edition:        config.Edition,
	}

	// Clear existing contexts if forcing
	if c.Force {
		abCfg.Contexts = []airbox.NamedContext{}
	}

	// Add the new context
	abCfg.AddContext(contextName, context)

	// Save to local file
	if err := cfg.Save(abCfg); err != nil {
		return fmt.Errorf("failed to save airbox config: %w", err)
	}

	ui.ShowInfo("Configuration saved to local airbox config!")
	ui.ShowKeyValue("Config file", airbox.GetConfigPath())
	ui.NewLine()

	return nil
}

// uiGuidedSetup provides interactive setup for configuration discovery
func (c *InitCmd) uiGuidedSetup(ctx context.Context, ui ui.Provider) (*abctl.Config, error) {
	// Step 1: Select deployment type
	deploymentOptions := []string{"Enterprise (SME)", "Cloud"}
	_, deploymentType, err := ui.Select("Select your Airbyte deployment type:", deploymentOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to select deployment type: %w", err)
	}

	switch deploymentType {
	case "Enterprise (SME)":
		return c.setupEnterprise(ctx, ui)
	case "Cloud":
		return c.setupCloud(ctx, ui)
	default:
		return nil, fmt.Errorf("unknown deployment type: %s", deploymentType)
	}
}

// setupEnterprise configures for SME deployment
func (c *InitCmd) setupEnterprise(ctx context.Context, ui ui.Provider) (*abctl.Config, error) {
	// Get SME endpoint from user
	endpoint, err := ui.TextInput("Enter your SME endpoint URL:", "https://your-sme.company.com", airbyte.ValidateEndpointURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get SME endpoint: %w", err)
	}

	instanceConfig, err := airbyte.ValidateEnterpriseEndpoint(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid enterprise endpoint: %w", err)
	}

	// Build configuration for SME
	config, err := airbyte.BuildEnterpriseConfig(endpoint, instanceConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build enterprise config: %w", err)
	}

	return config, nil
}

// setupCloud configures for Cloud deployment
func (c *InitCmd) setupCloud(_ context.Context, ui ui.Provider) (*abctl.Config, error) {
	// Ask about auth method

	methods := airbyte.DefaultCloudAuthMethods()
	idx, _, err := ui.Select("How do you sign in to Airbyte Cloud?", methods.Descriptions())
	if err != nil {
		return nil, fmt.Errorf("failed to select auth method: %w", err)
	}

	var realmIdentifier string
	cloudAuthMethod := methods[idx]
	if cloudAuthMethod.Name == airbyte.CloudAuthMethodSSO {
		// Get company identifier for SSO
		companyID, err := ui.TextInput("Enter your company identifier (from your SSO login URL):", "my-company", airbyte.ValidateCompanyIdentifier)
		if err != nil {
			return nil, fmt.Errorf("failed to get company identifier: %w", err)
		}
		realmIdentifier = companyID
	} else {
		// Use standard cloud users realm for email/password auth
		realmIdentifier = "_airbyte-cloud-users"
	}

	// Build configuration for Cloud
	config, err := airbyte.BuildCloudConfig(realmIdentifier, &cloudAuthMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to build cloud config: %w", err)
	}

	return config, nil
}
