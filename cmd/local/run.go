package local

import (
	"fmt"
	"github.com/airbytehq/abctl/internal/local"
	"github.com/airbytehq/abctl/internal/local/k8s"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
)

func runInstall(cmd *cobra.Command, _ []string) error {
	return telemetryWrapper(telemetry.Install, func() error {
		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("cluster - checking status of cluster %s", provider.ClusterName))

		cluster, err := k8s.NewCluster(provider)
		if err != nil {
			spinner.Fail(fmt.Sprintf("cluster - unable to determine status of cluster %s", provider.ClusterName))
			return err
		}

		if cluster.Exists() {
			spinner.Success(fmt.Sprintf("cluster - found existing cluster %s", provider.ClusterName))
		} else {
			spinner.UpdateText(fmt.Sprintf("cluster - creating cluster %s", provider.ClusterName))

			if err := cluster.Create(); err != nil {
				spinner.Fail(fmt.Sprintf("cluster - failed to create cluster %s", provider.ClusterName))
				return err
			}

			spinner.Success(fmt.Sprintf("cluster - cluster %s created", provider.ClusterName))
		}

		lc, err := local.New(provider, flagPortHTTP, local.WithTelemetryClient(telClient))
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

		return lc.Install(cmd.Context(), user, pass)
	})
}

func runUninstall(cmd *cobra.Command, _ []string) error {
	return telemetryWrapper(telemetry.Uninstall, func() error {
		spinnerClusterCheck, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("cluster - checking status of cluster %s", provider.ClusterName))

		cluster, err := k8s.NewCluster(provider)
		if err != nil {
			spinnerClusterCheck.Fail(fmt.Sprintf("cluster - unable to determine status of cluster %s", provider.ClusterName))
			return err
		}

		// if no cluster exists, there is nothing to do
		if !cluster.Exists() {
			spinnerClusterCheck.Success(fmt.Sprintf("cluster - unable to find existing cluster %s, nothing to uninstall", provider.ClusterName))
			return nil
		} else {
			spinnerClusterCheck.Success(fmt.Sprintf("cluster - found existing cluster %s", provider.ClusterName))
		}

		lc, err := local.New(provider, flagPortHTTP, local.WithTelemetryClient(telClient))
		if err != nil {
			pterm.Warning.Printfln("could not initialize local command: %s", err.Error())
			pterm.Warning.Println("will still attempt to uninstall the cluster")
		} else {
			if err := lc.Uninstall(cmd.Context()); err != nil {
				pterm.Warning.Printfln("could not complete uninstall: %s", err.Error())
				pterm.Warning.Println("will still attempt to uninstall the cluster")
			}
		}

		spinnerClusterDelete, _ := pterm.DefaultSpinner.Start("cluster - checking status of cluster deleted")

		if err := cluster.Delete(); err != nil {
			return fmt.Errorf("could not delete cluster %s", provider.ClusterName)
		}
		spinnerClusterDelete.Success(fmt.Sprintf("cluster %s - successfully deleted", provider.ClusterName))
		return nil
	})
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
