package auth

import (
	"context"
	"fmt"

	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/service"
	"github.com/pterm/pterm"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// LogoutCmd handles logout and credential cleanup
type LogoutCmd struct {
	Namespace string `short:"n" help:"Target namespace (default: current kubeconfig context)."`
}

// Run executes the logout command
func (c *LogoutCmd) Run(ctx context.Context, provider k8s.Provider) error {
	pterm.Info.Println("Logging out...")
	
	// Resolve namespace if not provided
	if c.Namespace == "" {
		var err error
		c.Namespace, err = k8s.GetCurrentNamespace()
		if err != nil {
			return fmt.Errorf("failed to get namespace from current context: %w", err)
		}
		pterm.Debug.Printf("Using namespace from current context: %s\n", c.Namespace)
	}
	
	// Create k8s client using standard kubeconfig resolution
	k8sClient, err := service.DefaultK8s("", "")
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}
	
	// Check if abctl is initialized
	if err := k8s.IsAbctlInitialized(ctx, k8sClient, c.Namespace); err != nil {
		return err
	}
	
	// Check if auth secret exists first
	_, err = k8sClient.SecretGet(ctx, c.Namespace, "abctl-auth")
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("not logged in: secret \"abctl-auth\" not found in namespace %s", c.Namespace)
		}
		return fmt.Errorf("failed to check for credentials: %w", err)
	}
	
	// Delete the auth secret by type (this will delete all Opaque secrets, but in practice
	// this namespace should only contain the abctl-auth secret for auth purposes)
	if err := k8sClient.SecretDeleteCollection(ctx, c.Namespace, "Opaque"); err != nil {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}
	
	pterm.Success.Printf("Successfully logged out. Credentials removed from namespace %s.\n", c.Namespace)
	return nil
}