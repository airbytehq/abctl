package local

import (
	"fmt"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/status"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	airbyteAuthSecretName = "airbyte-auth-secrets"
	airbyteNamespace      = "airbyte-abctl"
)

func NewCmdCredentials(provider k8s.Provider) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credentials",
		Short: "Get Airbyte user credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			return telClient.Wrap(cmd.Context(), telemetry.Install, func() error {

				client, err := defaultK8s(provider.Kubeconfig, provider.Context)
				if err != nil {
					status.Error("No existing cluster found")
					return nil
				}
				secret, err := client.SecretGet(cmd.Context(), airbyteNamespace, airbyteAuthSecretName)
				if err != nil {
					return err
				}

				//status.Success(fmt.Sprintf("Getting your credentials: %s", secret.Name))
				status.Info(fmt.Sprintf(
					"{\n  \"password\"      : \"%s\",\n  \"client-id\"     : \"%s\",\n  \"client-secret\" : \"%s\"\n}",
					secret.Data["instance-admin-password"],
					secret.Data["instance-admin-client-id"],
					secret.Data["instance-admin-client-secret"],
				))
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
