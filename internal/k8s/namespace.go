package k8s

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
)

// GetCurrentNamespace returns the namespace from the current kubeconfig context.
// If no namespace is set in the context, it returns "default".
func GetCurrentNamespace() (string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	)
	
	namespace, _, err := config.Namespace()
	if err != nil {
		return "", err
	}
	
	if namespace == "" {
		return "default", nil
	}
	
	return namespace, nil
}

// IsAbctlInitialized checks if abctl is initialized in the given namespace.
// Returns an error with helpful message if not initialized.
func IsAbctlInitialized(ctx context.Context, client Client, namespace string) error {
	_, err := client.ConfigMapGet(ctx, namespace, "abctl")
	if err != nil {
		return fmt.Errorf("abctl not initialized: %w (hint: run 'abctl init' first)", err)
	}
	return nil
}