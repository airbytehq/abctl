package local

import (
	"airbyte.io/abctl/internal/local"
	"airbyte.io/abctl/internal/telemetry"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Cmd represents the local command
var Cmd = &cobra.Command{
	Use: "local",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// ignore the error as it will default to false if an error returns
		dnt, _ := cmd.Flags().GetBool("dnt")

		var err error
		telClient, err = getTelemetryClient(dnt)
		if err != nil {
			// if the telemetry telClient fails to load, log a warning and continue
			pterm.Warning.Println(fmt.Errorf("unable to create telemetry telClient: %w", err))
		}

		return nil
	},
	Short: "Manages local Airbyte installations",
}

// installCmd is the local install command.
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Airbyte locally",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		return telemetryWrapper(telemetry.Install, func() error {
			lc, err := local.New(local.WithTelemetryClient(telClient))
			if err != nil {
				return fmt.Errorf("could not initialize local telClient: %w", err)
			}

			user := flagUsername
			if env := os.Getenv(envBasicAuthUser); env != "" {
				user = env
			}
			pass := flagPassword
			if env := os.Getenv(envBasicAuthPass); env != "" {
				pass = env
			}

			return lc.Install(context.Background(), user, pass)
		})
	},
}

// uninstallCmd is the local uninstall command.
var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Airbyte locally",
	RunE: func(cmd *cobra.Command, args []string) error {
		return telemetryWrapper(telemetry.Uninstall, func() error {
			lc, err := local.New(local.WithTelemetryClient(telClient))
			if err != nil {
				return fmt.Errorf("could not initialize local telClient: %w", err)
			}

			return lc.Uninstall(context.Background())
		})
	},
}

// telemetryWrapper wraps the function calls with the telemetry handlers
func telemetryWrapper(et telemetry.EventType, f func() error) (err error) {
	if err := telClient.Start(et); err != nil {
		pterm.Warning.Println("unable to send telemetry start data: %w", err)
	}
	defer func() {
		if err != nil {
			if err := telClient.Failure(et, err); err != nil {
				pterm.Warning.Println("unable to send telemetry failure data: %w", err)
			}
		} else {
			if err := telClient.Success(et); err != nil {
				pterm.Warning.Println("unable to send telemetry success data: %w", err)
			}
		}
	}()

	return f()
}

// getTelemetryClient fetches the telemetry telClient to use.
// If dnt (do-not-track) is true, this method will return a telemetry.NoopClient and will not attempt to read or
// write the telemetry.ConfigFile.
// If dnt is false, this method will read or write the telemetry.ConfigFile and will utilize an actual telemetry telClient.
func getTelemetryClient(dnt bool) (telemetry.Client, error) {
	if dnt {
		return telemetry.NoopClient{}, nil
	}
	cfg, err := getOrCreateCfg()
	if err != nil {
		return telemetry.NoopClient{}, fmt.Errorf("could not get or create config: %w", err)
	}
	return telemetry.NewSegmentClient(cfg), nil
}

// getOrCreateCfg returns the telemetry.Config data as read from telemetry.ConfigFile.
// If the telemetry.ConfigFile does not exist, this method will create it.
func getOrCreateCfg() (telemetry.Config, error) {
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
		println(telemetry.Welcome)
	} else if err != nil {
		return telemetry.Config{}, fmt.Errorf("could not load config from %s: %w", home, err)
	}

	return cfg, nil
}

var (
	// flagUsername is the basic-auth username as read from the cli flag
	flagUsername string
	// flagPassword is the basic-auth password as ready from the cli flag
	flagPassword string
	// flagProvider
	flagProvider string

	// telClient is the telemetry telClient to use
	telClient telemetry.Client
)

const (
	// envBasicAuthUser is the env-var that can be specified to override the default basic-auth username.
	envBasicAuthUser = "ABCTL_LOCAL_INSTALL_USERNAME"
	// envBasicAuthPass is the env-var that can be specified to override the default basic-auth password.
	envBasicAuthPass = "ABCTL_LOCAL_INSTALL_PASSWORD"
)

func init() {
	Cmd.PersistentFlags().StringVarP(&flagProvider, "k8s-cluster", "k", "docker-desktop", "k8s cluster")
	Cmd.AddCommand(installCmd)
	Cmd.AddCommand(uninstallCmd)
	installCmd.Flags().StringVarP(&flagUsername, "username", "u", "airbyte", "basic auth username, can also be specified via the env-var "+envBasicAuthUser)
	installCmd.Flags().StringVarP(&flagPassword, "password", "p", "password", "basic auth password, can also be specified via the env-var "+envBasicAuthPass)
}
