package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/service"
	"github.com/pterm/pterm"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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
	
	// Surgically remove just the "credentials" key from the secret data
	patch := map[string]interface{}{
		"data": map[string]interface{}{
			"credentials": nil,
		},
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to create patch: %w", err)
	}
	
	if err := k8sClient.SecretPatch(ctx, c.Namespace, "abctl-auth", patchBytes, types.StrategicMergePatchType); err != nil {
		return fmt.Errorf("failed to remove credentials: %w", err)
	}
	
	pterm.Success.Printf("Successfully logged out. Credentials removed from namespace %s.\n", c.Namespace)
	return nil
}