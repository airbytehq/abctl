package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/pterm/pterm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnsureValidAuth loads credentials and refreshes if expired
func EnsureValidAuth(ctx context.Context, k8sClient k8s.Client, namespace string) (*Credentials, error) {
	// Load credentials from secret
	secret, err := k8sClient.SecretGet(ctx, namespace, "abctl-auth")
	if err != nil {
		return nil, fmt.Errorf("not authenticated: %w (run 'abctl auth login')", err)
	}

	credData, ok := secret.Data["credentials"]
	if !ok {
		return nil, fmt.Errorf("credentials not found in secret")
	}

	creds, err := CredentialsFromJSON(credData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Check if token needs refresh (expired or expires within 5 minutes)
	if !needsRefresh(creds) {
		return creds, nil // Token is still valid
	}

	pterm.Debug.Println("Access token expired or expiring soon, refreshing...")

	// Refresh the token
	refreshed, err := refreshAuth(ctx, k8sClient, namespace, creds)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w (run 'abctl auth login' to re-authenticate)", err)
	}

	return refreshed, nil
}

// needsRefresh checks if credentials need refreshing
func needsRefresh(creds *Credentials) bool {
	// Refresh if token expires within 5 minutes
	bufferTime := 5 * time.Minute
	return time.Now().After(creds.ExpiresAt.Add(-bufferTime))
}

// refreshAuth refreshes the access token using the refresh token
func refreshAuth(ctx context.Context, k8sClient k8s.Client, namespace string, creds *Credentials) (*Credentials, error) {
	if creds.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	// Load abctl config to get auth URL
	config, err := k8s.GetAbctlConfig(ctx, k8sClient, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Discover provider
	provider, err := DiscoverProvider(ctx, config.AirbyteAuthURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover provider: %w", err)
	}

	// Refresh the token
	tokens, err := RefreshAccessToken(ctx, provider, DefaultClientID, creds.RefreshToken)
	if err != nil {
		return nil, err
	}

	// Calculate new expiration
	expiresAt := time.Now()
	if tokens.ExpiresIn > 0 {
		expiresAt = expiresAt.Add(time.Duration(tokens.ExpiresIn) * time.Second)
	} else {
		expiresAt = expiresAt.Add(time.Hour) // Default to 1 hour
	}

	// Update credentials
	newCreds := &Credentials{
		AccessToken:  tokens.AccessToken,
		RefreshToken: creds.RefreshToken, // Keep existing refresh token by default
		TokenType:    tokens.TokenType,
		ExpiresAt:    expiresAt,
	}

	// If a new refresh token was provided, use it
	if tokens.RefreshToken != "" {
		newCreds.RefreshToken = tokens.RefreshToken
	}

	// Store updated credentials
	if err := storeCredentials(ctx, k8sClient, namespace, newCreds); err != nil {
		return nil, fmt.Errorf("failed to store refreshed credentials: %w", err)
	}

	return newCreds, nil
}

// storeCredentials saves credentials to k8s secret
func storeCredentials(ctx context.Context, k8sClient k8s.Client, namespace string, creds *Credentials) error {
	credData, err := creds.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize credentials: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "abctl-auth",
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"credentials": credData,
		},
	}

	// Update the secret
	return k8sClient.SecretCreateOrUpdate(ctx, *secret)
}
