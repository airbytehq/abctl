package local

import (
	"context"
	"errors"
	"fmt"
	"github.com/airbytehq/abctl/internal/local"
	"github.com/airbytehq/abctl/internal/local/k8s"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

// telClient is the telemetry telClient to use
var telClient telemetry.Client

var provider k8s.Provider

// Cmd represents the local command
var Cmd = &cobra.Command{
	Use: "local",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
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

		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("cluster - checking status of cluster %s", k8s.ClusterName))

		cluster, err := k8s.NewCluster(provider)
		if err != nil {
			spinner.Fail(fmt.Sprintf("cluster - unable to determine status of cluster %s", k8s.ClusterName))
			return err
		}

		if cluster.Exists(k8s.ClusterName) {
			spinner.Success(fmt.Sprintf("cluster - found existing cluster %s", k8s.ClusterName))
		} else {
			spinner.UpdateText(fmt.Sprintf("cluster - creating cluster %s", k8s.ClusterName))

			if err := cluster.Create(k8s.ClusterName); err != nil {
				spinner.Fail(fmt.Sprintf("cluster - failed to create cluster %s", k8s.ClusterName))
				return err
			}

			spinner.Success(fmt.Sprintf("cluster - cluster %s created", k8s.ClusterName))
		}

		return nil
	},
	Short: "Manages local Airbyte installations",
}

func printK8sProvider(p k8s.Provider) {
	userHome, _ := os.UserHomeDir()
	configPath := filepath.Join(userHome, p.Kubeconfig)
	pterm.Info.Printfln("using kubernetes provider:\n  provider name: %s\n  kubeconfig: %s\n  context: %s",
		p.Name, configPath, p.Context)
}

const (
	// envBasicAuthUser is the env-var that can be specified to override the default basic-auth username.
	envBasicAuthUser = "ABCTL_LOCAL_INSTALL_USERNAME"
	// envBasicAuthPass is the env-var that can be specified to override the default basic-auth password.
	envBasicAuthPass = "ABCTL_LOCAL_INSTALL_PASSWORD"
)

// InstallCmd installs Airbyte locally
var InstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Airbyte locally",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		{
			userHome, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("could not determine user home directory: %w", err)
			}

			if err := dockerInstalled(cmd.Context(), telClient, userHome); err != nil {
				return fmt.Errorf("could not determine docker installation status: %w", err)
			}
		}

		if err := portAvailable(cmd.Context(), 80); err != nil {
			return fmt.Errorf("could not check available port: %w", err)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		return telemetryWrapper(telemetry.Install, func() error {
			lc, err := local.New(provider, local.WithTelemetryClient(telClient))
			if err != nil {
				return fmt.Errorf("could not initialize local command: %w", err)
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

// UninstallCmd uninstalls Airbyte locally
var UninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Airbyte locally",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return telemetryWrapper(telemetry.Uninstall, func() error {
			lc, err := local.New(provider, local.WithTelemetryClient(telClient))
			if err != nil {
				return fmt.Errorf("could not initialize local command: %w", err)
			}

			return lc.Uninstall(context.Background())
		})
	},
}

// telemetryWrapper wraps the function calls with the telemetry handlers
func telemetryWrapper(et telemetry.EventType, f func() error) (err error) {
	if err := telClient.Start(et); err != nil {
		pterm.Warning.Printfln("unable to send telemetry start data: %s", err)
	}
	defer func() {
		if err != nil {
			if err := telClient.Failure(et, err); err != nil {
				pterm.Warning.Printfln("unable to send telemetry failure data: %s", err)
			}
		} else {
			if err := telClient.Success(et); err != nil {
				pterm.Warning.Printfln("unable to send telemetry success data: %s", err)
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
			println(telemetry.Welcome)
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

var (
	flagUsername    string
	flagPassword    string
	flagProvider    string
	flagKubeconfig  string
	flagKubeContext string
)

func init() {
	InstallCmd.Flags().StringVarP(&flagUsername, "username", "u", "airbyte", "basic auth username, can also be specified via "+envBasicAuthUser)
	InstallCmd.Flags().StringVarP(&flagPassword, "password", "p", "password", "basic auth password, can also be specified via "+envBasicAuthPass)

	// switch the default provider based on the operating system... not sure if I like this idea or not
	defaultProvider := k8s.KindProvider.Name
	switch runtime.GOOS {
	case "darwin":
		defaultProvider = k8s.DockerDesktopProvider.Name
	case "windows":
		defaultProvider = k8s.DockerDesktopProvider.Name
	}

	Cmd.PersistentFlags().StringVarP(&flagProvider, "k8s-provider", "k", defaultProvider, "kubernetes provider to use")
	Cmd.PersistentFlags().StringVarP(&flagKubeconfig, "kubeconfig", "", "", "kubernetes config file to use")
	Cmd.PersistentFlags().StringVarP(&flagKubeContext, "kubecontext", "", "", "kubernetes context to use")
	Cmd.AddCommand(InstallCmd)
	Cmd.AddCommand(UninstallCmd)
}
