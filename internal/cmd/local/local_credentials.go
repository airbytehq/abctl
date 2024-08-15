package local

import (
	"fmt"

	"github.com/airbytehq/abctl/internal/cmd/local/airbyte"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	airbyteAuthSecretName = "airbyte-auth-secrets"
	airbyteNamespace      = "airbyte-abctl"

	secretPassword     = "instance-admin-password"
	secretClientID     = "instance-admin-client-id"
	secretClientSecret = "instance-admin-client-secret"
)

func NewCmdCredentials(provider k8s.Provider) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credentials",
		Short: "Get Airbyte user credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			return telClient.Wrap(cmd.Context(), telemetry.Install, func() error {

				client, err := defaultK8s(provider.Kubeconfig, provider.Context)
				if err != nil {
					pterm.Error.Println("No existing cluster found")
					return nil
				}
				secret, err := client.SecretGet(cmd.Context(), airbyteNamespace, airbyteAuthSecretName)
				if err != nil {
					return err
				}

				clientId := string(secret.Data[secretClientID])
				clientSecret := string(secret.Data[secretClientSecret])

				port, err := getPort(cmd.Context(), provider)
				if err != nil {
					return err
				}

				abAPI := airbyte.New(fmt.Sprintf("http://localhost:%d", port), clientId, clientSecret)
				orgEmail, err := abAPI.GetOrgEmail(cmd.Context())
				if err != nil {
					pterm.Error.Println("Unable to determine organization email")
					return fmt.Errorf("unable to determine organization email: %w", err)
				}

				pterm.Success.Println(fmt.Sprintf("Getting your credentials: %s", secret.Name))
				pterm.Info.Println(fmt.Sprintf(`Credentials:
  Email: %s
  Password: %s
  Client-Id: %s
  Client-Secret: %s`, orgEmail, secret.Data[secretPassword], clientId, clientSecret))
				return nil
			})
		},
	}

	return cmd
}

func defaultK8s(kubecfg, kubectx string) (k8s.Client, error) {
	k8sCfg, err := k8sClientConfig(kubecfg, kubectx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", localerr.ErrKubernetes, err)
	}

	restCfg, err := k8sCfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("%w: could not create rest config: %w", localerr.ErrKubernetes, err)
	}
	k8sClient, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("%w: could not create clientset: %w", localerr.ErrKubernetes, err)
	}

	return &k8s.DefaultK8sClient{ClientSet: k8sClient}, nil
}

// k8sClientConfig returns a k8s client config using the ~/.kube/config file and the k8sContext context.
func k8sClientConfig(kubecfg, kubectx string) (clientcmd.ClientConfig, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubecfg},
		&clientcmd.ConfigOverrides{CurrentContext: kubectx},
	), nil
}
