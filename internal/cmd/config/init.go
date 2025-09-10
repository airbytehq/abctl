package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/ui"
)

// InitCmd represents the init command
type InitCmd struct {
	Force bool `flag:"" help:"Overwrite existing airbox configuration."`
}

// Run executes the init command
func (c *InitCmd) Run(ctx context.Context, cfgStore airbox.ConfigStore, ui ui.Provider) error {
	ui.Title("Initializing airbox configuration")

	// Check if config already exists first
	if cfgStore.Exists() && !c.Force {
		return fmt.Errorf("airbox configuration already exists, use --force to overwrite")
	}

	// Prompt for edition
	_, edition, err := ui.Select("What Airbyte control plane are you connecting to:", []string{"Enterprise Flex", "Self-Managed Enterprise"})
	if err != nil {
		return fmt.Errorf("failed to get edition: %w", err)
	}

	var abCtx *airbox.Context
	var contextName string

	switch edition {
	case "Enterprise Flex":
		abCtx, contextName, err = c.setupCloud(ctx, ui)
	case "Self-Managed Enterprise":
		abCtx, contextName, err = c.setupEnterprise(ctx, ui)
	default:
		return fmt.Errorf("unsupported edition: %s", edition)
	}

	if err != nil {
		return err
	}

	// Create fresh config
	cfg := &airbox.Config{
		Contexts: []airbox.NamedContext{},
	}

	// Add the context
	cfg.AddContext(contextName, *abCtx)

	// Validate the entire config structure before saving
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Save to local file
	if err := cfgStore.Save(cfg); err != nil {
		return fmt.Errorf("failed to save airbox config: %w", err)
	}

	ui.ShowSuccess("Configuration saved successfully!")
	ui.NewLine()

	return nil
}

// setupCloud configures for Cloud deployment with OAuth only
func (c *InitCmd) setupCloud(ctx context.Context, ui ui.Provider) (*airbox.Context, string, error) {
	// Initialize the context that will hold our configuration
	abCtx := airbox.Context{
		Edition: "cloud",
	}

	// Get cloud URLs with env var override support for testing/staging environments
	cloudDomain := os.Getenv("AIRBYTE_CLOUD_DOMAIN")
	if cloudDomain == "" {
		cloudDomain = "cloud.airbyte.com"
	}

	cloudAPIDomain := os.Getenv("AIRBYTE_CLOUD_API_DOMAIN")
	if cloudAPIDomain == "" {
		cloudAPIDomain = "api.airbyte.com"
	}

	// Set the base URLs in our context
	abCtx.AirbyteAPIURL = fmt.Sprintf("https://%s", cloudAPIDomain)
	abCtx.AirbyteURL = fmt.Sprintf("https://%s", cloudDomain)

	// OAuth only - load client credentials from environment
	envCfg, err := airbox.LoadOAuthEnvConfig()
	if err != nil {
		return nil, "", fmt.Errorf("failed to load OAuth config: %w", err)
	}
	abCtx.Auth = airbox.NewAuthWithOAuth2(envCfg.ClientID, envCfg.ClientSecret)

	// Use the airbyteURL as the context name
	return &abCtx, abCtx.AirbyteURL, nil
}

// setupEnterprise configures for Enterprise edition with OAuth
func (c *InitCmd) setupEnterprise(ctx context.Context, ui ui.Provider) (*airbox.Context, string, error) {
	// Prompt for Airbyte URL
	airbyteURL, err := ui.TextInput("Enter your Airbyte instance URL (e.g., https://airbyte.yourcompany.com):", "", nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get Airbyte URL: %w", err)
	}

	// Remove trailing slash if present
	airbyteURL = strings.TrimSuffix(airbyteURL, "/")

	// API host is base URL + /api
	apiHost := airbyteURL + "/api"

	// Initialize the context
	abCtx := airbox.Context{
		AirbyteURL:    airbyteURL,
		AirbyteAPIURL: apiHost,
		Edition:       "enterprise",
	}

	// OAuth only - load client credentials from environment
	envCfg, err := airbox.LoadOAuthEnvConfig()
	if err != nil {
		return nil, "", fmt.Errorf("failed to load OAuth config: %w", err)
	}
	abCtx.Auth = airbox.NewAuthWithOAuth2(envCfg.ClientID, envCfg.ClientSecret)

	// Use the airbyteURL as the context name
	return &abCtx, airbyteURL, nil
}
