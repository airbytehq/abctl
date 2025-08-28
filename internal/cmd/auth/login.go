package auth

import (
	"context"
	"fmt"

	"github.com/airbytehq/abctl/internal/auth"
	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/service"
	"github.com/pterm/pterm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LoginCmd handles OAuth login flow
type LoginCmd struct {
	Namespace    string `short:"n" help:"Target namespace (default: current kubeconfig context)."`
	CallbackPort int    `flag:"" default:"8085" help:"Port for OAuth callback server."`
}

// Run executes the login command
func (c *LoginCmd) Run(ctx context.Context, provider k8s.Provider) error {
	pterm.Info.Println("Starting authentication flow...")
	
	// Resolve namespace if not provided
	if c.Namespace == "" {
		var err error
		c.Namespace, err = k8s.GetCurrentNamespace()
		if err != nil {
			return fmt.Errorf("failed to get namespace from current context: %w", err)
		}
		pterm.Debug.Printf("Using namespace from current context: %s\n", c.Namespace)
	}

	// Create k8s client - pass empty strings to use default kubeconfig resolution
	k8sClient, err := service.DefaultK8s("", "")
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	// Check if abctl is initialized
	if err := k8s.IsAbctlInitialized(ctx, k8sClient, c.Namespace); err != nil {
		return err
	}
	
	// Load abctl config
	config, err := k8s.GetAbctlConfig(ctx, k8sClient, c.Namespace)
	if err != nil {
		return err
	}

	pterm.Info.Printf("API Host: %s\n", config.AirbyteAPIHost)
	
	// Check if auth URL is configured
	if config.AirbyteAuthURL == "" {
		return fmt.Errorf("airbyteAuthURL not configured in abctl ConfigMap")
	}
	
	pterm.Info.Printf("Connecting to auth server: %s\n", config.AirbyteAuthURL)

	// Discover OIDC provider
	authProvider, err := auth.DiscoverProvider(ctx, config.AirbyteAuthURL)
	if err != nil {
		return fmt.Errorf("failed to discover auth provider: %w", err)
	}

	// Create OAuth flow with custom port
	flow := auth.NewFlow(authProvider, auth.DefaultClientID, c.CallbackPort)

	// Perform authentication
	credentials, err := flow.Authenticate(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Store credentials in k8s secret
	if err := c.storeCredentials(ctx, k8sClient, credentials); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	pterm.Success.Println("Successfully authenticated! Credentials stored securely.")
	return nil
}

func (c *LoginCmd) storeCredentials(ctx context.Context, client k8s.Client, creds *auth.Credentials) error {
	credData, err := creds.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize credentials: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "abctl-auth",
			Namespace: c.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"credentials": credData,
		},
	}

	// Try to create, if exists then update
	if err := client.SecretCreateOrUpdate(ctx, *secret); err != nil {
		return fmt.Errorf("failed to store credentials secret: %w", err)
	}

	return nil
}
