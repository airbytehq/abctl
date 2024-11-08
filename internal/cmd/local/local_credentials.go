package local

import (
	"context"
	"fmt"

	"github.com/airbytehq/abctl/internal/cmd/local/airbyte"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/local"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
	"go.opencensus.io/trace"
)

const (
	airbyteAuthSecretName = "airbyte-auth-secrets"
	airbyteNamespace      = "airbyte-abctl"

	secretPassword     = "instance-admin-password"
	secretClientID     = "instance-admin-client-id"
	secretClientSecret = "instance-admin-client-secret"
)

type CredentialsCmd struct {
	Email    string `help:"Specify a new email address to use for authentication."`
	Password string `help:"Specify a new password to use for authentication."`
}

func (cc *CredentialsCmd) Run(ctx context.Context, provider k8s.Provider, telClient telemetry.Client) error {
	ctx, span := trace.StartSpan(ctx, "local credentials")
	defer span.End()

	spinner := &pterm.DefaultSpinner

	return telClient.Wrap(ctx, telemetry.Credentials, func() error {
		k8sClient, err := local.DefaultK8s(provider.Kubeconfig, provider.Context)
		if err != nil {
			pterm.Error.Println("No existing cluster found")
			return nil
		}

		secret, err := k8sClient.SecretGet(ctx, airbyteNamespace, airbyteAuthSecretName)
		if err != nil {
			return err
		}

		clientId := string(secret.Data[secretClientID])
		clientSecret := string(secret.Data[secretClientSecret])

		port, err := getPort(ctx, provider.ClusterName)
		if err != nil {
			return err
		}

		abAPI := airbyte.New(fmt.Sprintf("http://localhost:%d", port), clientId, clientSecret)

		if cc.Email != "" {
			pterm.Info.Println("Updating email for authentication")
			if err := abAPI.SetOrgEmail(ctx, cc.Email); err != nil {
				pterm.Error.Println("Unable to update the email address")
				return fmt.Errorf("unable to udpate the email address: %w", err)
			}
			pterm.Success.Println("Email updated")
		}

		if cc.Password != "" && cc.Password != string(secret.Data[secretPassword]) {
			pterm.Info.Println("Updating password for authentication")
			secret.Data[secretPassword] = []byte(cc.Password)
			if err := k8sClient.SecretCreateOrUpdate(ctx, *secret); err != nil {
				pterm.Error.Println("Unable to update the password")
				return fmt.Errorf("unable to update the password: %w", err)
			}
			pterm.Success.Println("Password updated")

			// as the secret was updated, fetch it again
			secret, err = k8sClient.SecretGet(ctx, airbyteNamespace, airbyteAuthSecretName)
			if err != nil {
				return err
			}

			spinner, _ = spinner.Start("Restarting airbyte-abctl-server")
			if err := k8sClient.DeploymentRestart(ctx, airbyteNamespace, "airbyte-abctl-server"); err != nil {
				pterm.Error.Println("Unable to restart airbyte-abctl-server")
				return fmt.Errorf("unable to restart airbyte-abctl-server: %w", err)
			}
			spinner.Success("Restarted airbyte-abctl-server")
		}

		orgEmail, err := abAPI.GetOrgEmail(ctx)
		if err != nil {
			pterm.Error.Println("Unable to determine organization email")
			return fmt.Errorf("unable to determine organization email: %w", err)
		}
		if orgEmail == "" {
			orgEmail = "[not set]"
		}

		pterm.Success.Println(fmt.Sprintf("Retrieving your credentials from '%s'", secret.Name))
		pterm.Info.Println(fmt.Sprintf(`Credentials:
  Email: %s
  Password: %s
  Client-Id: %s
  Client-Secret: %s`, orgEmail, secret.Data[secretPassword], clientId, clientSecret))
		return nil
	})
}
