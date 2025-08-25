package init

import (
	"context"
	"fmt"
	"time"

	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/service"
	"github.com/pterm/pterm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Cmd represents the init command
type Cmd struct {
	Namespace     string `flag:"" short:"n" help:"Target c.Namespace (default: current kubeconfig context)."`
	Force         bool   `flag:"" help:"Overwrite existing abctl ConfigMap."`
	FromConfigmap string `flag:"" help:"Source ConfigMap name (default: auto-detect via airbyte=templates label)."`
}

// Run executes the init command
func (c *Cmd) Run(ctx context.Context, provider k8s.Provider) error {
	pterm.Info.Println("Initializing abctl configuration...")

	pterm.Info.Printf("Using c.Namespace: %s\n", c.Namespace)
	pterm.Debug.Printf("Provider kubeconfig: %s\n", provider.Kubeconfig)
	pterm.Debug.Printf("Provider context: %s\n", provider.Context)

	// Create k8s client using standard kubeconfig resolution (KUBECONFIG env var or ~/.kube/config)
	k8sClient, err := service.DefaultK8s("", "")
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	// Determine source ConfigMap name
	sourceConfigMapName := c.FromConfigmap
	if sourceConfigMapName == "" {
		// Find ConfigMap with -airbyte-env suffix
		sourceConfigMapName, err = findAirbyteEnvConfigMap(ctx, k8sClient, c.Namespace)
		if err != nil {
			return fmt.Errorf("failed to auto-detect Airbyte ConfigMap: %w", err)
		}
	}

	pterm.Info.Printf("Reading from ConfigMap: %s\n", sourceConfigMapName)

	// Read the source ConfigMap
	pterm.Debug.Printf("Attempting to get ConfigMap: c.Namespace=%s, name=%s\n", c.Namespace, sourceConfigMapName)
	sourceConfigMap, err := k8sClient.ConfigMapGet(ctx, c.Namespace, sourceConfigMapName)
	if err != nil {
		return fmt.Errorf("failed to read ConfigMap %s/%s: %w", c.Namespace, sourceConfigMapName, err)
	}

	pterm.Success.Printf("Found ConfigMap %s with %d keys\n", sourceConfigMapName, len(sourceConfigMap.Data))

	// Extract key configuration values
	config, err := k8s.AbctlConfigFromData(sourceConfigMap.Data)
	if err != nil {
		return fmt.Errorf("failed to extract configuration: %w", err)
	}

	pterm.Info.Printf("Extracted configuration:\n")
	pterm.Info.Printf("  Airbyte API Host: %s\n", config.AirbyteAPIHost)
	pterm.Info.Printf("  Airbyte URL: %s\n", config.AirbyteURL)

	// Check if abctl ConfigMap already exists
	const abctlConfigMapName = "abctl"
	_, err = k8sClient.ConfigMapGet(ctx, c.Namespace, abctlConfigMapName)
	if err == nil && !c.Force {
		return fmt.Errorf("abctl ConfigMap already exists in c.Namespace %s, use --force to overwrite", c.Namespace)
	}

	// Create abctl ConfigMap
	abctlConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      abctlConfigMapName,
			Namespace: c.Namespace,
			Annotations: map[string]string{
				"abctl.airbyte.com/initialized-from": sourceConfigMapName,
				"abctl.airbyte.com/initialized-at":   time.Now().Format(time.RFC3339),
			},
		},
		Data: map[string]string{
			"airbyteApiHost": config.AirbyteAPIHost,
			"airbyteURL":     config.AirbyteURL,
			"airbyteAuthURL": config.AirbyteAuthURL,
		},
	}

	// Create or update the ConfigMap
	if err = k8sClient.ConfigMapCreate(ctx, abctlConfigMap); err != nil {
		// If creation failed and --force is set, try to update
		if c.Force {
			if updateErr := k8sClient.ConfigMapUpdate(ctx, abctlConfigMap); updateErr != nil {
				return fmt.Errorf("failed to create or update abctl ConfigMap: %w", updateErr)
			}
			pterm.Success.Printf("Updated abctl ConfigMap in c.Namespace %s\n", c.Namespace)
		} else {
			return fmt.Errorf("failed to create abctl ConfigMap: %w", err)
		}
	} else {
		pterm.Success.Printf("Created abctl ConfigMap in c.Namespace %s\n", c.Namespace)
	}

	pterm.Info.Println("Configuration initialization completed successfully")

	return nil
}

// findAirbyteEnvConfigMap finds a ConfigMap whose name ends with "-airbyte-env" suffix
func findAirbyteEnvConfigMap(ctx context.Context, k8sClient k8s.Client, namespace string) (string, error) {
	configMaps, err := k8sClient.ConfigMapList(ctx, namespace)
	if err != nil {
		return "", fmt.Errorf("failed to list ConfigMaps: %w", err)
	}

	const suffix = "-airbyte-env"
	for _, cm := range configMaps.Items {
		if len(cm.Name) >= len(suffix) && cm.Name[len(cm.Name)-len(suffix):] == suffix {
			return cm.Name, nil
		}
	}

	return "", fmt.Errorf("no ConfigMap with suffix %q found in namespace %s", suffix, namespace)
}
