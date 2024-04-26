package local

import (
	"errors"
	"fmt"
	"github.com/airbytehq/abctl/internal/local/k8s"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

// persistentPreRunLocal is extracted from the cobra.Command initialization for reading purposes only
func persistentPreRunLocal(cmd *cobra.Command, _ []string) error {
	// telemetry client configuration
	{
		// ignore the error as it will default to false if an error returns
		dnt, _ := cmd.Flags().GetBool("dnt")

		var err error
		telClient, err = getTelemetryClient(dnt)
		if err != nil {
			// if the telemetry telClient fails to load, log a warning and continue
			pterm.Warning.Println(fmt.Errorf("unable to create telemetry telClient: %w", err))
		}
	}
	// provider configuration
	{
		var err error
		provider, err = k8s.ProviderFromString(flagProvider)
		if err != nil {
			return err
		}

		printK8sProvider(provider)
	}

	return nil
}

func printK8sProvider(p k8s.Provider) {
	userHome, _ := os.UserHomeDir()
	configPath := filepath.Join(userHome, p.Kubeconfig)
	pterm.Info.Printfln("using kubernetes provider:\n  provider name: %s\n  kubeconfig: %s\n  context: %s",
		p.Name, configPath, p.Context)
}

// preRunInstall is extracted from the cobra.Command initialization for reading purposes only
func preRunInstall(cmd *cobra.Command, _ []string) error {
	if err := dockerInstalled(cmd.Context(), telClient); err != nil {
		return fmt.Errorf("could not determine docker installation status: %w", err)
	}

	if err := portAvailable(cmd.Context(), flagPort); err != nil {
		return fmt.Errorf("port %d is not available: %w", flagPort, err)
	}

	return nil
}

// preRunUninstall is extracted from the cobra.Command initialization for reading purposes only
func preRunUninstall(cmd *cobra.Command, _ []string) error {
	if err := dockerInstalled(cmd.Context(), telClient); err != nil {
		return fmt.Errorf("could not determine docker installation status: %w", err)
	}

	return nil
}

// getTelemetryClient fetches the telemetry telClient to use.
// If dnt (do-not-track) is true, this method will return a telemetry.NoopClient and will not attempt to read or
// write the telemetry.ConfigFile.
// If dnt is false, this method will read or write the telemetry.ConfigFile and will utilize an actual telemetry telClient.
func getTelemetryClient(dnt bool) (telemetry.Client, error) {
	if dnt {
		return telemetry.NoopClient{}, nil
	}

	// getOrCreateCfg returns the telemetry.Config data as read from telemetry.ConfigFile.
	// If the telemetry.ConfigFile does not exist, this function will create it.
	getOrCreateCfg := func() (telemetry.Config, error) {
		home, err := os.UserHomeDir()
		if err != nil {
			return telemetry.Config{}, fmt.Errorf("could not locate home directory: %w", err)
		}
		home = filepath.Join(home, telemetry.ConfigFile)
		cfg, err := telemetry.LoadConfigFromFile(home)
		if errors.Is(err, os.ErrNotExist) {
			// file not found, create a new one
			cfg = telemetry.Config{UserID: telemetry.NewULID()}
			if err := telemetry.WriteConfigToFile(home, cfg); err != nil {
				return cfg, fmt.Errorf("could not write file to %s: %w", home, err)
			}
			pterm.Info.Println(telemetry.Welcome)
		} else if err != nil {
			return telemetry.Config{}, fmt.Errorf("could not load config from %s: %w", home, err)
		}

		return cfg, nil
	}

	cfg, err := getOrCreateCfg()
	if err != nil {
		return telemetry.NoopClient{}, fmt.Errorf("could not get or create config: %w", err)
	}
	return telemetry.NewSegmentClient(cfg), nil
}
