package local

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/airbytehq/abctl/internal/cmd/local/docker"
	"github.com/airbytehq/abctl/internal/cmd/local/helm"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s/kind"
	"github.com/airbytehq/abctl/internal/cmd/local/migrate"
	"github.com/airbytehq/abctl/internal/cmd/local/paths"
	"github.com/airbytehq/abctl/internal/maps"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"

	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/cli/browser"
	"github.com/google/uuid"
	helmclient "github.com/mittwald/go-helm-client"
	"github.com/mittwald/go-helm-client/values"
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/repo"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	airbyteChartName    = "airbyte/airbyte"
	airbyteChartRelease = "airbyte-abctl"
	airbyteIngress      = "ingress-abctl"
	airbyteNamespace    = "airbyte-abctl"
	airbyteRepoName     = "airbyte"
	airbyteRepoURL      = "https://airbytehq.github.io/helm-charts"
	nginxChartName      = "nginx/ingress-nginx"
	nginxChartRelease   = "ingress-nginx"
	nginxNamespace      = "ingress-nginx"
	nginxRepoName       = "nginx"
	nginxRepoURL        = "https://kubernetes.github.io/ingress-nginx"
)

// dockerAuthSecretName is the name of the secret which holds the docker authentication information.
const dockerAuthSecretName = "docker-auth"

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// BrowserLauncher primarily for testing purposes.
type BrowserLauncher func(url string) error

// Command is the local command, responsible for installing, uninstalling, or other local actions.
type Command struct {
	provider k8s.Provider
	cluster  k8s.Cluster
	http     HTTPClient
	helm     helm.Client
	k8s      k8s.Client
	portHTTP int
	spinner  *pterm.SpinnerPrinter
	tel      telemetry.Client
	launcher BrowserLauncher
	userHome string
}

// Option for configuring the Command, primarily exists for testing
type Option func(*Command)

// WithTelemetryClient define the telemetry client for this command.
func WithTelemetryClient(client telemetry.Client) Option {
	return func(c *Command) {
		c.tel = client
	}
}

// WithHTTPClient define the http client for this command.
func WithHTTPClient(client HTTPClient) Option {
	return func(c *Command) {
		c.http = client
	}
}

// WithHelmClient define the helm client for this command.
func WithHelmClient(client helm.Client) Option {
	return func(c *Command) {
		c.helm = client
	}
}

// WithK8sClient define the k8s client for this command.
func WithK8sClient(client k8s.Client) Option {
	return func(c *Command) {
		c.k8s = client
	}
}

// WithBrowserLauncher define the browser launcher for this command.
func WithBrowserLauncher(launcher BrowserLauncher) Option {
	return func(c *Command) {
		c.launcher = launcher
	}
}

// WithUserHome define the user's home directory.
func WithUserHome(home string) Option {
	return func(c *Command) {
		c.userHome = home
	}
}

func WithSpinner(spinner *pterm.SpinnerPrinter) Option {
	return func(c *Command) {
		c.spinner = spinner
	}
}

func WithPortHTTP(port int) Option {
	return func(c *Command) {
		c.portHTTP = port
	}
}

// New creates a new Command
func New(provider k8s.Provider, opts ...Option) (*Command, error) {
	c := &Command{provider: provider}
	for _, opt := range opts {
		opt(c)
	}

	// determine userhome if not defined
	if c.userHome == "" {
		c.userHome = paths.UserHome
	}

	// set http client, if not defined
	if c.http == nil {
		c.http = &http.Client{Timeout: 10 * time.Second}
	}

	if c.portHTTP == 0 {
		c.portHTTP = kind.IngressPort
	}

	// set k8s client, if not defined
	if c.k8s == nil {
		var err error
		if c.k8s, err = defaultK8s(provider.Kubeconfig, provider.Context); err != nil {
			return nil, err
		}
	}

	// set the helm client, if not defined
	if c.helm == nil {
		var err error
		if c.helm, err = helm.New(provider.Kubeconfig, provider.Context, airbyteNamespace); err != nil {
			return nil, err
		}
	}

	// set telemetry client, if not defined
	if c.tel == nil {
		c.tel = telemetry.NoopClient{}
	}

	// set spinner, if not defined
	if c.spinner == nil {
		c.spinner, _ = pterm.DefaultSpinner.Start()
	}

	// set the browser launcher, if not defined
	if c.launcher == nil {
		c.launcher = browser.OpenURL
	}

	// fetch k8s version information
	{
		k8sVersion, err := c.k8s.ServerVersionGet()
		if err != nil {
			return nil, fmt.Errorf("%w: unable to fetch kubernetes server version: %w", localerr.ErrKubernetes, err)
		}
		c.tel.Attr("k8s_version", k8sVersion)
	}

	// set provider version
	c.tel.Attr("provider", provider.Name)

	return c, nil
}

type InstallOpts struct {
	HelmChartVersion string
	ValuesFile       string
	Secrets          []string
	Migrate          bool
	Host             string

	Docker *docker.Docker

	DockerServer string
	DockerUser   string
	DockerPass   string
	DockerEmail  string

	NoBrowser       bool
	LowResourceMode bool
	InsecureCookies bool
}

func (i *InstallOpts) dockerAuth() bool {
	return i.DockerUser != "" && i.DockerPass != ""
}

const (
	// persistent volume constants, these are named to match the values given in the helm chart
	pvMinio = "airbyte-minio-pv"
	pvPsql  = "airbyte-volume-db"

	// persistent volume claim constants, these are named to match the values given in the helm chart
	pvcMinio = "airbyte-minio-pv-claim-airbyte-minio-0"
	pvcPsql  = "airbyte-volume-db-airbyte-db-0"
)

// persistentVolume creates a persistent volume in the namespace with the name provided.
// if uid (user id) and gid (group id) are non-zero, the persistent directory on the host machine that holds the
// persistent volume will be changed to be owned by
func (c *Command) persistentVolume(ctx context.Context, namespace, name string) error {
	if !c.k8s.PersistentVolumeExists(ctx, namespace, name) {
		c.spinner.UpdateText(fmt.Sprintf("Creating persistent volume '%s'", name))

		// Pre-create the volume directory.
		//
		// K8s, when using HostPathDirectoryOrCreate will create the directory (if it doesn't exist)
		// with 0755 permissions _but_ it will be owned by the user under which the docker daemon is running,
		// not the user that is running this code.
		//
		// This causes issues if the docker daemon is running as root but this code is not (the expectation is
		// that this code should not need to run as root), as the non-root user will not have write permissions to
		// k8s created directory (again 0755 permissions).
		//
		// By pre-creating the volume directory we can ensure that the owner of that directory will be the
		// user that is running this code and not the user that is running the docker daemon.
		path := filepath.Join(paths.Data, name)

		pterm.Debug.Println(fmt.Sprintf("Creating directory '%s'", path))
		if err := os.MkdirAll(path, 0766); err != nil {
			pterm.Error.Println(fmt.Sprintf("Unable to create directory '%s'", name))
			return fmt.Errorf("unable to create persistent volume '%s': %w", name, err)
		}

		if err := c.k8s.PersistentVolumeCreate(ctx, namespace, name); err != nil {
			pterm.Error.Println(fmt.Sprintf("Unable to create persistent volume '%s'", name))
			return fmt.Errorf("unable to create persistent volume '%s': %w", name, err)
		}

		// Update the permissions of the volume directory to be globally writable (0777).
		//
		// This is necessary to ensure that the postgres image can actually write to the volume directory
		// that it is assigned, as the postgres image creates its own user (postgres:postgres) with
		// uid/gid of 70:70.  If this uid/gid doesn't exist on the host machine, then the postgres image
		// will not be able to write to the volume directory unless that directory is publicly writable.
		//
		// Why not set the permissions to 0777 when this directory was created earlier in this method?
		// Because it is likely that the host has a umask defined that would override this 0777 to 0775 or 0755.
		// Due to the postgres uid/gid issue mentioned above, 0775 or 0755 would not allow the postgres image
		// access to the persisted volume directory.
		pterm.Debug.Println(fmt.Sprintf("Updating permissions for '%s'", path))
		if err := os.Chmod(path, 0777); err != nil {
			pterm.Error.Println(fmt.Sprintf("Unable to set permissions for '%s'", path))
			return fmt.Errorf("unable to set permissions for '%s': %w", path, err)
		}

		pterm.Info.Println(fmt.Sprintf("Persistent volume '%s' created", name))
	} else {
		pterm.Info.Printfln("Persistent volume '%s' already exists", name)
	}

	return nil
}

func (c *Command) persistentVolumeClaim(ctx context.Context, namespace, name, volumeName string) error {
	if !c.k8s.PersistentVolumeClaimExists(ctx, namespace, name, volumeName) {
		c.spinner.UpdateText(fmt.Sprintf("Creating persistent volume claim '%s'", name))
		if err := c.k8s.PersistentVolumeClaimCreate(ctx, namespace, name, volumeName); err != nil {
			pterm.Error.Println(fmt.Sprintf("Unable to create persistent volume claim '%s'", name))
			return fmt.Errorf("unable to create persistent volume claim '%s': %w", name, err)
		}
		pterm.Info.Println(fmt.Sprintf("Persistent volume claim '%s' created", name))
	} else {
		pterm.Info.Printfln("Persistent volume claim '%s' already exists", name)
	}

	return nil
}

// Install handles the installation of Airbyte
func (c *Command) Install(ctx context.Context, opts InstallOpts) error {
	go c.watchEvents(ctx)

	if !c.k8s.NamespaceExists(ctx, airbyteNamespace) {
		c.spinner.UpdateText(fmt.Sprintf("Creating namespace '%s'", airbyteNamespace))
		if err := c.k8s.NamespaceCreate(ctx, airbyteNamespace); err != nil {
			pterm.Error.Println(fmt.Sprintf("Unable to create namespace '%s'", airbyteNamespace))
			return fmt.Errorf("unable to create airbyte namespace: %w", err)
		}
		pterm.Info.Println(fmt.Sprintf("Namespace '%s' created", airbyteNamespace))
	} else {
		pterm.Info.Printfln("Namespace '%s' already exists", airbyteNamespace)
	}

	if err := c.persistentVolume(ctx, airbyteNamespace, pvMinio); err != nil {
		return err
	}
	if err := c.persistentVolume(ctx, airbyteNamespace, pvPsql); err != nil {
		return err
	}

	if opts.Migrate {
		c.spinner.UpdateText("Migrating airbyte data")
		//if err := c.tel.Wrap(ctx, telemetry.Migrate, func() error { return opts.Docker.MigrateComposeDB(ctx, "airbyte_db") }); err != nil {
		if err := c.tel.Wrap(ctx, telemetry.Migrate, func() error { return migrate.FromDockerVolume(ctx, opts.Docker.Client, "airbyte_db") }); err != nil {
			pterm.Error.Println("Failed to migrate data from previous Airbyte installation")
			return fmt.Errorf("unable to migrate data from previous airbyte installation: %w", err)
		}
	}

	if err := c.persistentVolumeClaim(ctx, airbyteNamespace, pvcMinio, pvMinio); err != nil {
		return err
	}
	if err := c.persistentVolumeClaim(ctx, airbyteNamespace, pvcPsql, pvPsql); err != nil {
		return err
	}

	var telUser string
	// only override the empty telUser if the tel.User returns a non-nil (uuid.Nil) value.
	if c.tel.User() != uuid.Nil {
		telUser = c.tel.User().String()
	}

	airbyteValues := []string{
		"global.env_vars.AIRBYTE_INSTALLATION_ID=" + telUser,
		"global.auth.enabled=true",
	}

	if opts.LowResourceMode {
		airbyteValues = append(airbyteValues,
			"server.env_vars.JOB_RESOURCE_VARIANT_OVERRIDE=lowresource",
			"server.env_vars.JOB_MAIN_CONTAINER_CPU_LIMIT=0",
			"server.env_vars.JOB_MAIN_CONTAINER_CPU_REQUEST=0",
			"server.env_vars.JOB_MAIN_CONTAINER_MEMORY_LIMIT=0",
			"server.env_vars.JOB_MAIN_CONTAINER_MEMORY_REQUEST=0",

			"workload-launcher.env_vars.JOB_MAIN_CONTAINER_CPU_LIMIT=0",
			"workload-launcher.env_vars.JOB_MAIN_CONTAINER_CPU_REQUEST=0",
			"workload-launcher.env_vars.JOB_MAIN_CONTAINER_MEMORY_LIMIT=0",
			"workload-launcher.env_vars.JOB_MAIN_CONTAINER_MEMORY_REQUEST=0",
			"workload-launcher.env_vars.CHECK_JOB_MAIN_CONTAINER_CPU_LIMIT=0",
			"workload-launcher.env_vars.CHECK_JOB_MAIN_CONTAINER_CPU_REQUEST=0",
			"workload-launcher.env_vars.CHECK_JOB_MAIN_CONTAINER_MEMORY_LIMIT=0",
			"workload-launcher.env_vars.CHECK_JOB_MAIN_CONTAINER_MEMORY_REQUEST=0",
			"workload-launcher.env_vars.DISCOVER_JOB_MAIN_CONTAINER_CPU_LIMIT=0",
			"workload-launcher.env_vars.DISCOVER_JOB_MAIN_CONTAINER_CPU_REQUEST=0",
			"workload-launcher.env_vars.DISCOVER_JOB_MAIN_CONTAINER_MEMORY_LIMIT=0",
			"workload-launcher.env_vars.DISCOVER_JOB_MAIN_CONTAINER_MEMORY_REQUEST=0",
			"workload-launcher.env_vars.SPEC_JOB_MAIN_CONTAINER_CPU_LIMIT=0",
			"workload-launcher.env_vars.SPEC_JOB_MAIN_CONTAINER_CPU_REQUEST=0",
			"workload-launcher.env_vars.SPEC_JOB_MAIN_CONTAINER_MEMORY_LIMIT=0",
			"workload-launcher.env_vars.SPEC_JOB_MAIN_CONTAINER_MEMORY_REQUEST=0",
			"workload-launcher.env_vars.SIDECAR_MAIN_CONTAINER_CPU_LIMIT=0",
			"workload-launcher.env_vars.SIDECAR_MAIN_CONTAINER_CPU_REQUEST=0",
			"workload-launcher.env_vars.SIDECAR_MAIN_CONTAINER_MEMORY_LIMIT=0",
			"workload-launcher.env_vars.SIDECAR_MAIN_CONTAINER_MEMORY_REQUEST=0",
		)
	} else {
		airbyteValues = append(airbyteValues,
			"global.jobs.resources.limits.cpu=3",
			"global.jobs.resources.limits.memory=4Gi",
		)
	}
	if opts.InsecureCookies {
		airbyteValues = append(airbyteValues,
			"global.auth.cookieSecureSetting=false")
	}

	if opts.dockerAuth() {
		pterm.Debug.Println(fmt.Sprintf("Creating '%s' secret", dockerAuthSecretName))
		if err := c.handleDockerSecret(ctx, opts.DockerServer, opts.DockerUser, opts.DockerPass, opts.DockerEmail); err != nil {
			pterm.Debug.Println(fmt.Sprintf("Unable to create '%s' secret", dockerAuthSecretName))
			return fmt.Errorf("unable to create '%s' secret: %w", dockerAuthSecretName, err)
		}
		pterm.Debug.Println(fmt.Sprintf("Created '%s' secret", dockerAuthSecretName))
		airbyteValues = append(airbyteValues, fmt.Sprintf("global.imagePullSecrets[0].name=%s", dockerAuthSecretName))
	}

	for _, secretFile := range opts.Secrets {
		c.spinner.UpdateText(fmt.Sprintf("Creating secret from '%s'", secretFile))
		raw, err := os.ReadFile(secretFile)
		if err != nil {
			pterm.Error.Println(fmt.Sprintf("Unable to read secret file '%s': %s", secretFile, err))
			return fmt.Errorf("unable to read secret file '%s': %w", secretFile, err)
		}

		var secret corev1.Secret
		if err := yaml.Unmarshal(raw, &secret); err != nil {
			pterm.Error.Println(fmt.Sprintf("Unable to unmarshal secret file '%s': %s", secretFile, err))
			return fmt.Errorf("unable to unmarshal secret file '%s': %w", secretFile, err)
		}
		secret.ObjectMeta.Namespace = airbyteNamespace

		if err := c.k8s.SecretCreateOrUpdate(ctx, secret); err != nil {
			pterm.Error.Println(fmt.Sprintf("Unable to create secret from file '%s'", secretFile))
			return fmt.Errorf("unable to create secret from file '%s': %w", secretFile, err)
		}

		pterm.Success.Println(fmt.Sprintf("Secret from '%s' created or updated", secretFile))
	}

	valuesYAML, err := mergeValuesWithValuesYAML(airbyteValues, opts.ValuesFile)
	if err != nil {
		return fmt.Errorf("unable to merge values with values file '%s': %w", opts.ValuesFile, err)
	}

	if err := c.handleChart(ctx, chartRequest{
		name:         "airbyte",
		repoName:     airbyteRepoName,
		repoURL:      airbyteRepoURL,
		chartName:    airbyteChartName,
		chartRelease: airbyteChartRelease,
		chartVersion: opts.HelmChartVersion,
		namespace:    airbyteNamespace,
		valuesYAML:   valuesYAML,
	}); err != nil {
		return fmt.Errorf("unable to install airbyte chart: %w", err)
	}

	if err := c.handleChart(ctx, chartRequest{
		name:           "nginx",
		uninstallFirst: true,
		repoName:       nginxRepoName,
		repoURL:        nginxRepoURL,
		chartName:      nginxChartName,
		chartRelease:   nginxChartRelease,
		namespace:      nginxNamespace,
		values:         append(c.provider.HelmNginx, fmt.Sprintf("controller.service.ports.http=%d", c.portHTTP)),
	}); err != nil {
		// If we timed out, there is a good chance it's due to an unavailable port, check if this is the case.
		// As the kubernetes client doesn't return usable error types, have to check for a specific string value.
		if strings.Contains(err.Error(), "client rate limiter Wait returned an error") {
			pterm.Warning.Printfln("Encountered an error while installing the %s Helm Chart.\n"+
				"This could be an indication that port %d is not available.\n"+
				"If installation fails, please try again with a different port.", nginxChartName, c.portHTTP)

			srv, err := c.k8s.ServiceGet(ctx, nginxNamespace, "ingress-nginx-controller")
			// If there is an error, we can ignore it as we only are checking for a missing ingress entry,
			// and an error would indicate the inability to check for that entry.
			if err == nil {
				ingresses := srv.Status.LoadBalancer.Ingress
				if len(ingresses) == 0 {
					// if there are no ingresses, that is a possible indicator that the port is already in use.
					return fmt.Errorf("%w: could not install nginx chart", localerr.ErrIngress)
				}
			}
		}
		return fmt.Errorf("unable to install nginx chart: %w", err)
	}

	if err := c.handleIngress(ctx, opts.Host); err != nil {
		return err
	}

	// verify ingress using localhost
	url := fmt.Sprintf("http://localhost:%d", c.portHTTP)
	if err := c.verifyIngress(ctx, url); err != nil {
		return err
	}

	if opts.NoBrowser {
		pterm.Success.Println(fmt.Sprintf(
			"Launching web-browser disabled. Airbyte should be accessible at\n  %s",
			url,
		))
	} else {
		c.launch(url)
	}

	return nil
}

func (c *Command) handleIngress(ctx context.Context, host string) error {
	c.spinner.UpdateText("Checking for existing Ingress")

	if c.k8s.IngressExists(ctx, airbyteNamespace, airbyteIngress) {
		pterm.Success.Println("Found existing Ingress")
		if err := c.k8s.IngressUpdate(ctx, airbyteNamespace, ingress(host)); err != nil {
			pterm.Error.Printfln("Unable to update existing Ingress")
			return fmt.Errorf("unable to update existing ingress: %w", err)
		}
		pterm.Success.Println("Updated existing Ingress")
		return nil
	}

	pterm.Info.Println("No existing Ingress found, creating one")
	if err := c.k8s.IngressCreate(ctx, airbyteNamespace, ingress(host)); err != nil {
		pterm.Error.Println("Unable to create ingress")
		return fmt.Errorf("unable to create ingress: %w", err)
	}
	pterm.Success.Println("Ingress created")
	return nil
}

func (c *Command) watchEvents(ctx context.Context) {
	watcher, err := c.k8s.EventsWatch(ctx, airbyteNamespace)
	if err != nil {
		pterm.Warning.Printfln("Unable to watch airbyte events\n  %s", err)
		return
	}
	defer watcher.Stop()

	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				pterm.Debug.Println("Event watcher completed.")
				return
			}
			if convertedEvent, ok := event.Object.(*eventsv1.Event); ok {
				c.handleEvent(ctx, convertedEvent)
			} else {
				pterm.Debug.Printfln("Received unexpected event: %T", event.Object)
			}
		case <-ctx.Done():
			pterm.Debug.Printfln("Event watcher context completed:\n  %s", ctx.Err())
			return
		}
	}
}

// now is used to filter out kubernetes events that happened in the past.
// Kubernetes wants us to use the ResourceVersion on the event watch request itself, but that approach
// is more complicated as it requires determining which ResourceVersion to initially provide.
var now = func() *metav1.Time {
	t := metav1.Now()
	return &t
}()

// handleEvent converts a kubernetes event into a console log message
func (c *Command) handleEvent(ctx context.Context, e *eventsv1.Event) {
	// TODO: replace DeprecatedLastTimestamp,
	// this is supposed to be replaced with series.lastObservedTime, however that field is always nil...
	if e.DeprecatedLastTimestamp.Before(now) {
		return
	}

	switch {
	case strings.EqualFold(e.Type, "normal"):
		pterm.Debug.Println(e.Note)
	case strings.EqualFold(e.Type, "warning"):
		var logs = ""
		if strings.EqualFold(e.Reason, "backoff") {
			var err error
			logs, err = c.k8s.LogsGet(ctx, e.Regarding.Namespace, e.Regarding.Name)
			if err != nil {
				pterm.Debug.Printfln("Unable to retrieve logs for %s:%s\n  %s", e.Regarding.Namespace, e.Regarding.Name, err)
			}
		}

		// TODO: replace DeprecatedCount
		// Similar issue to DeprecatedLastTimestamp, the series attribute is always nil
		if logs != "" {
			msg := fmt.Sprintf("Encountered an issue deploying Airbyte:\n  Pod: %s\n  Reason: %s\n  Message: %s\n  Count: %d\n  Logs: %s",
				e.Name, e.Reason, e.Note, e.DeprecatedCount, strings.TrimSpace(logs))
			pterm.Debug.Println(msg)
			// only show the warning if the count is higher than 5
			if e.DeprecatedCount > 5 {
				pterm.Warning.Printfln(msg)
			}
		} else {
			msg := fmt.Sprintf("Encountered an issue deploying Airbyte:\n  Pod: %s\n  Reason: %s\n  Message: %s\n  Count: %d",
				e.Name, e.Reason, e.Note, e.DeprecatedCount)
			pterm.Debug.Printfln(msg)
			// only show the warning if the count is higher than 5
			if e.DeprecatedCount > 5 {
				pterm.Warning.Printfln(msg)
			}
		}

	default:
		pterm.Debug.Printfln("Received an unsupported event type: %s", e.Type)
	}
}

func (c *Command) handleDockerSecret(ctx context.Context, server, user, pass, email string) error {
	secretBody, err := docker.Secret(server, user, pass, email)
	if err != nil {
		pterm.Error.Println("Unable to create docker secret")
		return fmt.Errorf("unable to create docker secret: %w", err)
	}

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: airbyteNamespace,
			Name:      dockerAuthSecretName,
		},
		Data: map[string][]byte{corev1.DockerConfigJsonKey: secretBody},
		Type: corev1.SecretTypeDockerConfigJson,
	}

	if err := c.k8s.SecretCreateOrUpdate(ctx, secret); err != nil {
		pterm.Error.Println("Unable to create Docker-auth secret")
		return fmt.Errorf("unable to create docker-auth secret: %w", err)
	}
	pterm.Success.Println("Docker-Auth secret created")
	return nil
}

type UninstallOpts struct {
	Persisted bool
}

// Uninstall handles the uninstallation of Airbyte.
func (c *Command) Uninstall(_ context.Context, opts UninstallOpts) error {
	// check if persisted data should be removed, if not this is a noop
	if opts.Persisted {
		c.spinner.UpdateText("Removing persisted data")
		if err := os.RemoveAll(paths.Data); err != nil {
			pterm.Error.Println(fmt.Sprintf("Unable to remove persisted data '%s'", paths.Data))
			return fmt.Errorf("unable to remove persisted data '%s': %w", paths.Data, err)
		}
		pterm.Success.Println("Removed persisted data")
	}

	return nil
}

// Status handles the status of local Airbyte.
func (c *Command) Status(_ context.Context) error {
	charts := []string{airbyteChartRelease, nginxChartRelease}
	for _, name := range charts {
		c.spinner.UpdateText(fmt.Sprintf("Verifying %s Helm Chart installation status", name))

		rel, err := c.helm.GetRelease(name)
		if err != nil {
			pterm.Warning.Println("Unable to fetch airbyte release")
			pterm.Debug.Printfln("unable to fetch airbyte release: %s", err)
			continue
		}

		pterm.Info.Println(fmt.Sprintf(
			"Found helm chart '%s'\n  Status: %s\n  Chart Version: %s\n  App Version: %s",
			name, rel.Info.Status.String(), rel.Chart.Metadata.Version, rel.Chart.Metadata.AppVersion,
		))
	}

	pterm.Info.Println(fmt.Sprintf("Airbyte should be accessible via http://localhost:%d", c.portHTTP))

	return nil
}

// chartRequest exists to make all the parameters to handleChart somewhat manageable
type chartRequest struct {
	name           string
	repoName       string
	repoURL        string
	chartName      string
	chartRelease   string
	chartVersion   string
	namespace      string
	values         []string
	valuesYAML     string
	uninstallFirst bool
}

// handleChart will handle the installation of a chart
func (c *Command) handleChart(
	ctx context.Context,
	req chartRequest,
) error {
	c.spinner.UpdateText(fmt.Sprintf("Configuring %s Helm repository", req.name))

	if err := c.helm.AddOrUpdateChartRepo(repo.Entry{
		Name: req.repoName,
		URL:  req.repoURL,
	}); err != nil {
		pterm.Error.Printfln("Unable to configure %s Helm repository", req.repoName)
		return fmt.Errorf("unable to add %s chart repo: %w", req.name, err)
	}

	c.spinner.UpdateText(fmt.Sprintf("Fetching %s Helm Chart", req.chartName))
	helmChart, _, err := c.helm.GetChart(req.chartName, &action.ChartPathOptions{Version: req.chartVersion})
	if err != nil {
		pterm.Error.Printfln("Unable to fetch %s Helm Chart", req.chartName)
		return fmt.Errorf("unable to fetch chart %s: %w", req.chartName, err)
	}

	c.tel.Attr(fmt.Sprintf("helm_%s_chart_version", req.name), helmChart.Metadata.Version)

	if req.uninstallFirst {
		action := determineHelmChartAction(c.helm, helmChart, req.chartRelease)
		switch action {
		case none:
			pterm.Success.Println(fmt.Sprintf(
				"Found matching existing Helm Chart %s:\n  Name: %s\n  Namespace: %s\n  Version: %s\n  AppVersion: %s",
				req.chartName, req.chartName, req.namespace, helmChart.Metadata.Version, helmChart.Metadata.AppVersion,
			))
			return nil
		case uninstall:
			pterm.Debug.Println(fmt.Sprintf("Attempting to uninstall Helm Chart %s", req.chartRelease))
			if err := c.helm.UninstallReleaseByName(req.chartRelease); err != nil {
				pterm.Error.Println(fmt.Sprintf("Unable to uninstall Helm Chart %s", req.chartRelease))
				return fmt.Errorf("unable to uninstall Helm Chart %s: %w", req.chartRelease, err)
			} else {
				pterm.Debug.Println(fmt.Sprintf("Uninstalled Helm Chart %s", req.chartRelease))
			}
		case install:
			pterm.Debug.Println(fmt.Sprintf("Will only attempt to install Helm Chart %s", req.chartRelease))
		default:
			pterm.Debug.Println(fmt.Sprintf("Unexpected response %d", action))
		}
	}

	pterm.Info.Println(fmt.Sprintf(
		"Starting Helm Chart installation of '%s' (version: %s)",
		req.chartName, helmChart.Metadata.Version,
	))
	c.spinner.UpdateText(fmt.Sprintf(
		"Installing '%s' (version: %s) Helm Chart (this may take several minutes)",
		req.chartName, helmChart.Metadata.Version,
	))
	helmRelease, err := c.helm.InstallOrUpgradeChart(ctx, &helmclient.ChartSpec{
		ReleaseName:     req.chartRelease,
		ChartName:       req.chartName,
		CreateNamespace: true,
		Namespace:       req.namespace,
		Wait:            true,
		Timeout:         30 * time.Minute,
		ValuesOptions:   values.Options{Values: req.values},
		ValuesYaml:      req.valuesYAML,
		Version:         req.chartVersion,
	},
		&helmclient.GenericHelmOptions{},
	)
	if err != nil {
		pterm.Error.Printfln("Failed to install %s Helm Chart", req.chartName)
		return fmt.Errorf("unable to install helm: %w", err)
	}

	c.tel.Attr(fmt.Sprintf("helm_%s_release_version", req.name), strconv.Itoa(helmRelease.Version))

	pterm.Success.Println(fmt.Sprintf(
		"Installed Helm Chart %s:\n  Name: %s\n  Namespace: %s\n  Version: %s\n  AppVersion: %s\n  Release: %d",
		req.chartName, helmRelease.Name, helmRelease.Namespace, helmRelease.Chart.Metadata.Version, helmRelease.Chart.Metadata.AppVersion, helmRelease.Version,
	))
	return nil
}

// verifyIngress will open the url in the user's browser but only if the url returns a 200 response code first
// TODO: clean up this method, make it testable
func (c *Command) verifyIngress(ctx context.Context, url string) error {
	c.spinner.UpdateText("Verifying ingress")

	ingressCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	alive := make(chan error)

	go func() {
		tick := time.Tick(1 * time.Second)
		for {
			select {
			case <-ingressCtx.Done():
				alive <- fmt.Errorf("liveness check failed: %w", ingressCtx.Err())
			case <-tick:
				req, err := http.NewRequestWithContext(ingressCtx, http.MethodGet, url, nil)
				if err != nil {
					alive <- fmt.Errorf("unable to create request: %w", err)
				}
				res, _ := c.http.Do(req)
				// if no auth, we should get a 200
				if res != nil && res.StatusCode == http.StatusOK {
					alive <- nil
				}
				// if basic auth, we should get a 401 with a specific header that contains abctl
				if res != nil && res.StatusCode == http.StatusUnauthorized && strings.Contains(res.Header.Get("WWW-Authenticate"), "abctl") {
					alive <- nil
				}
			}
		}
	}()

	select {
	case <-ingressCtx.Done():
		pterm.Error.Println("Timed out waiting for ingress")
		return fmt.Errorf("browser liveness check failed: %w", ingressCtx.Err())
	case err := <-alive:
		if err != nil {
			pterm.Error.Println("Ingress verification failed")
			return fmt.Errorf("browser failed liveness check: %w", err)
		}
	}
	// if we're here, then no errors occurred
	return nil
}

func (c *Command) launch(url string) {
	c.spinner.UpdateText(fmt.Sprintf("Attempting to launch web-browser for %s", url))

	if err := c.launcher(url); err != nil {
		pterm.Warning.Println(fmt.Sprintf(
			"Failed to launch web-browser.\nPlease launch your web-browser to access %s",
			url,
		))
		pterm.Debug.Println(fmt.Sprintf("failed to launch web-browser: %s", err.Error()))
		// don't consider a failed web-browser to be a failed installation
		return
	}

	pterm.Success.Println(fmt.Sprintf("Launched web-browser successfully for %s", url))
}

// defaultK8s returns the default k8s client
func defaultK8s(kubecfg, kubectx string) (k8s.Client, error) {
	rest.SetDefaultWarningHandler(k8s.Logger{})
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

type helmReleaseAction int

const (
	none helmReleaseAction = iota
	install
	uninstall
)

// determineHelmChartAction determines the state of the existing chart compared
// to what chart is being considered for installation.
//
// returns none if no additional action needs to be taken, uninstall if the chart exists and the
// version differs or not.
func determineHelmChartAction(helm helm.Client, chart *chart.Chart, releaseName string) helmReleaseAction {
	// look for an existing release, see if it matches the existing chart
	rel, err := helm.GetRelease(releaseName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			// chart hasn't been installed previously
			pterm.Debug.Println(fmt.Sprintf("Unable to find %s Helm Release", releaseName))
			return install
		} else {
			// chart may or may not exist, log error and ignore
			pterm.Debug.Println(fmt.Sprintf("Unable to fetch %s Helm Release: %s", releaseName, err))
			return uninstall
		}
	}

	if rel.Info.Status != release.StatusDeployed {
		pterm.Debug.Println(fmt.Sprintf("Chart has the status of %s", rel.Info.Status))
		return uninstall
	}

	if rel.Chart.Metadata.Version != chart.Metadata.Version {
		pterm.Debug.Println(fmt.Sprintf(
			"Chart version (%s) does not match Helm Release (%s)",
			chart.Metadata.Version, rel.Chart.Metadata.Version,
		))
		return uninstall
	}

	if rel.Chart.Metadata.AppVersion != chart.Metadata.AppVersion {
		pterm.Debug.Println(fmt.Sprintf(
			"Chart app-version (%s) does not match Helm Release (%s)",
			chart.Metadata.AppVersion, rel.Chart.Metadata.AppVersion,
		))
		return uninstall
	}

	pterm.Debug.Println(fmt.Sprintf(
		"Chart matched Helm Release\n  Version: %s - %s\n  AppVersion: %s - %s",
		chart.Metadata.Version, rel.Chart.Metadata.Version,
		chart.Metadata.AppVersion, rel.Chart.Metadata.AppVersion,
	))

	return none
}

// mergeValuesWithValuesYAML ensures that the values defined within this code have a lower
// priority than any values defined in a values.yaml file.
// By default, the helm-client we're using reversed this priority, putting the values
// defined in this code at a higher priority than the values defined in the values.yaml file.
// This function returns a string representation of the value.yaml file after all
// values provided were potentially overridden by the valuesYML file.
func mergeValuesWithValuesYAML(values []string, valuesYAML string) (string, error) {
	a := maps.FromSlice(values)
	b, err := maps.FromYAMLFile(valuesYAML)
	if err != nil {
		return "", fmt.Errorf("unable to read values from yaml file '%s': %w", valuesYAML, err)
	}
	maps.Merge(a, b)

	res, err := maps.ToYAML(a)
	if err != nil {
		return "", fmt.Errorf("unable to merge values from yaml file '%s': %w", valuesYAML, err)
	}

	return res, nil

}
