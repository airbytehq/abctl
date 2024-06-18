package local

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/cli/browser"
	"github.com/google/uuid"
	helmclient "github.com/mittwald/go-helm-client"
	"github.com/mittwald/go-helm-client/values"
	"github.com/pterm/pterm"
	"golang.org/x/crypto/bcrypt"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage/driver"
	corev1 "k8s.io/api/core/v1"
	v1events "k8s.io/api/events/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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

// Port is the default port that Airbyte will deploy to.
const Port = 8000

// HelmClient primarily for testing purposes
type HelmClient interface {
	AddOrUpdateChartRepo(entry repo.Entry) error
	GetChart(string, *action.ChartPathOptions) (*chart.Chart, string, error)
	GetRelease(name string) (*release.Release, error)
	InstallOrUpgradeChart(ctx context.Context, spec *helmclient.ChartSpec, opts *helmclient.GenericHelmOptions) (*release.Release, error)
	UninstallReleaseByName(string) error
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// BrowserLauncher primarily for testing purposes.
type BrowserLauncher func(url string) error

// Command is the local command, responsible for installing, uninstalling, or other local actions.
type Command struct {
	provider         k8s.Provider
	cluster          k8s.Cluster
	http             HTTPClient
	helm             HelmClient
	k8s              k8s.Client
	portHTTP         int
	spinner          *pterm.SpinnerPrinter
	tel              telemetry.Client
	launcher         BrowserLauncher
	userHome         string
	helmChartVersion string
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
func WithHelmClient(client HelmClient) Option {
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

func WithHelmChartVersion(version string) Option {
	return func(c *Command) {
		c.helmChartVersion = version
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
		var err error
		if c.userHome, err = os.UserHomeDir(); err != nil {
			return nil, fmt.Errorf("could not determine user home directory: %w", err)
		}
	}

	// set http client, if not defined
	if c.http == nil {
		c.http = &http.Client{Timeout: 10 * time.Second}
	}

	if c.portHTTP == 0 {
		c.portHTTP = Port
	}

	// set k8s client, if not defined
	if c.k8s == nil {
		kubecfg := filepath.Join(c.userHome, provider.Kubeconfig)
		var err error
		if c.k8s, err = defaultK8s(kubecfg, provider.Context); err != nil {
			return nil, err
		}
	}

	// set the helm client, if not defined
	if c.helm == nil {
		kubecfg := filepath.Join(c.userHome, provider.Kubeconfig)
		var err error
		if c.helm, err = defaultHelm(kubecfg, provider.Context); err != nil {
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

	if c.helmChartVersion == "latest" {
		c.helmChartVersion = ""
	}

	// fetch k8s version information
	{
		k8sVersion, err := c.k8s.ServerVersionGet()
		if err != nil {
			return nil, fmt.Errorf("%w: could not fetch kubernetes server version: %w", localerr.ErrKubernetes, err)
		}
		c.tel.Attr("k8s_version", k8sVersion)
	}

	// set provider version
	c.tel.Attr("provider", provider.Name)

	return c, nil
}

func pv(namespace, name string) *corev1.PersistentVolume {
	size, _ := resource.ParseQuantity("500Mi")
	hostPathType := corev1.HostPathDirectoryOrCreate

	return &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{corev1.ResourceStorage: size},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: path.Join("/var/local-path-provisioner", name),
					Type: &hostPathType,
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			PersistentVolumeReclaimPolicy: "Retain",
			StorageClassName:              "standard",
		},
	}
}

func pvc(name string, volumeName string) *corev1.PersistentVolumeClaim {
	size, _ := resource.ParseQuantity("500Mi")
	storageClass := "standard"

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources:        corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: size}},
			VolumeName:       volumeName,
			StorageClassName: &storageClass,
		},
		Status: corev1.PersistentVolumeClaimStatus{},
	}
}

// Install handles the installation of Airbyte
func (c *Command) Install(ctx context.Context, user, pass, valuesFile string) error {
	var values string
	if valuesFile != "" {
		raw, err := os.ReadFile(valuesFile)
		if err != nil {
			return fmt.Errorf("could not read values file '%s': %w", valuesFile, err)
		}
		values = string(raw)
	}

	go c.watchEvents(ctx)

	if !c.k8s.NamespaceExists(ctx, airbyteNamespace) {
		c.spinner.UpdateText(fmt.Sprintf("Creating namespace '%s'", airbyteNamespace))
		if err := c.k8s.NamespaceCreate(ctx, airbyteNamespace); err != nil {
			pterm.Error.Printfln("Could not create namespace '%s'", airbyteNamespace)
			return fmt.Errorf("could not create airbyte namespace: %w", err)
		}
	} else {
		pterm.Info.Printfln("Namespace '%s' already exists", airbyteNamespace)
	}

	// Create minio PV and PVC
	if _, err := c.k8s.TestClientSet().CoreV1().PersistentVolumes().Create(ctx, pv(airbyteNamespace, "airbyte-minio-pv"), metav1.CreateOptions{}); err != nil {
		pterm.Error.Printfln("Failed to create persistent volume: %s", err.Error())
	}

	if _, err := c.k8s.TestClientSet().CoreV1().PersistentVolumeClaims(airbyteNamespace).Create(ctx, pvc("airbyte-minio-pv-claim-airbyte-minio-0", "airbyte-minio-pv"), metav1.CreateOptions{}); err != nil {
		pterm.Error.Printfln("Failed to create persistent volume claim: %s", err.Error())
	}

	// Create database PV and PVC
	if _, err := c.k8s.TestClientSet().CoreV1().PersistentVolumes().Create(ctx, pv(airbyteNamespace, "airbyte-volume-db"), metav1.CreateOptions{}); err != nil {
		pterm.Error.Printfln("Failed to create persistent volume: %s", err.Error())
	}

	if _, err := c.k8s.TestClientSet().CoreV1().PersistentVolumeClaims(airbyteNamespace).Create(ctx, pvc("airbyte-volume-db-airbyte-db-0", "airbyte-volume-db"), metav1.CreateOptions{}); err != nil {
		pterm.Error.Printfln("Failed to create persistent volume claim: %s", err.Error())
	}

	var telUser string
	// only override the empty telUser if the tel.User returns a non-nil (uuid.Nil) value.
	if c.tel.User() != uuid.Nil {
		telUser = c.tel.User().String()
	}

	if err := c.handleChart(ctx, chartRequest{
		name:         "airbyte",
		repoName:     airbyteRepoName,
		repoURL:      airbyteRepoURL,
		chartName:    airbyteChartName,
		chartRelease: airbyteChartRelease,
		chartVersion: c.helmChartVersion,
		namespace:    airbyteNamespace,
		values:       []string{fmt.Sprintf("global.env_vars.AIRBYTE_INSTALLATION_ID=%s", telUser)},
		valuesYAML:   values,
	}); err != nil {
		return fmt.Errorf("could not install airbyte chart: %w", err)
	}

	if err := c.handleChart(ctx, chartRequest{
		name:         "nginx",
		repoName:     nginxRepoName,
		repoURL:      nginxRepoURL,
		chartName:    nginxChartName,
		chartRelease: nginxChartRelease,
		namespace:    nginxNamespace,
		values:       append(c.provider.HelmNginx, fmt.Sprintf("controller.service.ports.http=%d", c.portHTTP)),
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
		return fmt.Errorf("could not install nginx chart: %w", err)
	}

	c.spinner.UpdateText("Configuring Basic-Auth")
	// basic auth
	if err := c.handleBasicAuthSecret(ctx, user, pass); err != nil {
		return fmt.Errorf("could not create or update basic-auth secret: %w", err)
	}

	if err := c.handleIngress(ctx); err != nil {
		return err
	}

	c.spinner.UpdateText("Verifying ingress")
	if err := c.openBrowser(ctx, fmt.Sprintf("http://localhost:%d", c.portHTTP)); err != nil {
		return err
	}

	return nil
}

func (c *Command) handleIngress(ctx context.Context) error {
	c.spinner.UpdateText("Checking for existing Ingress")

	if c.k8s.IngressExists(ctx, airbyteNamespace, airbyteIngress) {
		pterm.Success.Println("Found existing Ingress")
		if err := c.k8s.IngressUpdate(ctx, airbyteNamespace, ingress()); err != nil {
			pterm.Error.Printfln("Unable to update existing Ingress")
			return fmt.Errorf("could not update existing ingress: %w", err)
		}
		pterm.Success.Println("Updated existing Ingress")
		return nil
	}

	pterm.Info.Println("No existing Ingress found, creating one")
	if err := c.k8s.IngressCreate(ctx, airbyteNamespace, ingress()); err != nil {
		pterm.Error.Println("Unable to create ingress")
		return fmt.Errorf("could not create ingress: %w", err)
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
			if convertedEvent, ok := event.Object.(*v1events.Event); ok {
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
func (c *Command) handleEvent(ctx context.Context, e *v1events.Event) {
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
			pterm.Warning.Printfln(
				"Encountered an issue deploying Airbyte:\n  Pod: %s\n  Reason: %s\n  Message: %s\n  Count: %d\n  Logs: %s",
				e.Name, e.Reason, e.Note, e.DeprecatedCount, strings.TrimSpace(logs),
			)
		} else {
			pterm.Warning.Printfln(
				"Encountered an issue deploying Airbyte:\n  Pod: %s\n  Reason: %s\n  Message: %s\n  Count: %d",
				e.Name, e.Reason, e.Note, e.DeprecatedCount,
			)
		}

	default:
		pterm.Debug.Printfln("Received an unsupported event type: %s", e.Type)
	}
}

// handleBasicAuthSecret creates or updates the appropriate basic auth credentials for ingress.
func (c *Command) handleBasicAuthSecret(ctx context.Context, user, pass string) error {
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		pterm.Error.Println("Basic Auth secret could not be hashed.\n" +
			"This may indicate an issue with the username or password provided.\n" +
			"Please provider different credentials and try again.")

		return fmt.Errorf("could not hash basic auth password: %w", err)
	}

	data := map[string][]byte{"auth": []byte(fmt.Sprintf("%s:%s", user, hashedPass))}
	if err := c.k8s.SecretCreateOrUpdate(ctx, airbyteNamespace, "basic-auth", data); err != nil {
		pterm.Error.Println("Could not create Basic-Auth secret")
	}
	pterm.Success.Println("Basic-Auth secret created")
	return nil
}

// Uninstall handles the uninstallation of Airbyte.
func (c *Command) Uninstall(ctx context.Context, persist bool) error {
	// if not removing persisted data, then this is a no-op
	if !persist {
		return nil
	}

	{
		c.spinner.UpdateText(fmt.Sprintf("Verifying %s Helm Chart installation status", airbyteChartName))

		airbyteChartExists := true
		if _, err := c.helm.GetRelease(airbyteChartRelease); err != nil {
			if !errors.Is(err, driver.ErrReleaseNotFound) {
				pterm.Error.Printfln("Could not verify installation status of %s Helm Chart", airbyteChartName)
				return fmt.Errorf("could not fetch airbyte release: %w", err)
			}

			pterm.Success.Printfln("Helm Chart %s is not installed", airbyteChartName)
			airbyteChartExists = false
		} else {
			pterm.Success.Printfln("Verified Helm Chart %s is installed", airbyteChartName)
		}

		if airbyteChartExists {
			c.spinner.UpdateText(fmt.Sprintf("Uninstalling %s Helm Chart", airbyteChartName))
			if err := c.helm.UninstallReleaseByName(airbyteChartRelease); err != nil {
				pterm.Error.Printfln("Could not uninstall %s Helm Chart", airbyteChartName)
				return fmt.Errorf("could not uninstall airbyte chart: %w", err)
			}
			pterm.Success.Printfln("Uninstalled %s Helm Chart", airbyteChartName)
		}
	}

	{
		c.spinner.UpdateText(fmt.Sprintf("Verifying %s Helm Chart installation status", nginxChartName))

		nginxChartExists := true
		if _, err := c.helm.GetRelease(nginxChartRelease); err != nil {
			if !errors.Is(err, driver.ErrReleaseNotFound) {
				pterm.Error.Printfln("Could not verify installation status of %s Helm Chart", nginxChartName)
				return fmt.Errorf("could not fetch nginx release: %w", err)
			}

			pterm.Success.Printfln("Helm Chart %s is not installed", nginxChartName)
			nginxChartExists = false
		}

		if nginxChartExists {
			c.spinner.UpdateText(fmt.Sprintf("Uninstalling %s Helm Chart", nginxChartName))
			if err := c.helm.UninstallReleaseByName(nginxChartRelease); err != nil {
				pterm.Error.Printfln("Could not uninstall %s Helm Chart", nginxChartName)
				return fmt.Errorf("could not uninstall nginx chart: %w", err)
			}
		}
		pterm.Success.Printfln("Uninstalled %s Helm Chart", nginxChartName)
	}

	c.spinner.UpdateText(fmt.Sprintf("Deleting Kubernetes namespace '%s'", airbyteNamespace))

	if err := c.k8s.NamespaceDelete(ctx, airbyteNamespace); err != nil {
		if !k8serrors.IsNotFound(err) {
			pterm.Error.Printfln("Could not delete Kubernetes namespace '%s'", airbyteNamespace)
			return fmt.Errorf("could not delete namespace: %w", err)
		}
	}

	// there is no blocking delete namespace call, so poll until it's been deleted, or we've exhausted our time
	namespaceDeleted := false
	var wg sync.WaitGroup
	ticker := time.NewTicker(1 * time.Second) // how ofter to check
	timer := time.After(5 * time.Minute)      // how long to wait
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ticker.C:
				if !c.k8s.NamespaceExists(ctx, airbyteNamespace) {
					namespaceDeleted = true
					return
				}
			case <-timer:
				ticker.Stop()
				return
			}
		}
	}()

	wg.Wait()

	if !namespaceDeleted {
		pterm.Error.Printfln("Could not delete Kubernetes namespace '%s'", airbyteNamespace)
		return errors.New("could not delete namespace")
	}

	pterm.Success.Printfln("Namespace '%s' deleted", airbyteNamespace)

	return nil
}

// Status handles the status of local Airbyte.
func (c *Command) Status(_ context.Context) error {
	charts := []string{airbyteChartRelease, nginxChartRelease}
	for _, name := range charts {
		c.spinner.UpdateText(fmt.Sprintf("Verifying %s Helm Chart installation status", name))

		rel, err := c.helm.GetRelease(name)
		if err != nil {
			pterm.Warning.Println("Could not get airbyte release")
			pterm.Debug.Printfln("could not get airbyte release: %s", err)
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
	name         string
	repoName     string
	repoURL      string
	chartName    string
	chartRelease string
	chartVersion string
	namespace    string
	values       []string
	valuesYAML   string
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
		return fmt.Errorf("could not add %s chart repo: %w", req.name, err)
	}

	c.spinner.UpdateText(fmt.Sprintf("Fetching %s Helm Chart", req.chartName))
	helmChart, _, err := c.helm.GetChart(req.chartName, &action.ChartPathOptions{Version: req.chartVersion})
	if err != nil {
		pterm.Error.Printfln("Unable to fetch %s Helm Chart", req.chartName)
		return fmt.Errorf("could not fetch chart %s: %w", req.chartName, err)
	}

	c.tel.Attr(fmt.Sprintf("helm_%s_chart_version", req.name), helmChart.Metadata.Version)

	c.spinner.UpdateText(fmt.Sprintf("Installing '%s' (version: %s) Helm Chart", req.chartName, helmChart.Metadata.Version))
	helmRelease, err := c.helm.InstallOrUpgradeChart(ctx, &helmclient.ChartSpec{
		ReleaseName:     req.chartRelease,
		ChartName:       req.chartName,
		CreateNamespace: true,
		Namespace:       req.namespace,
		Wait:            true,
		Timeout:         10 * time.Minute,
		ValuesOptions:   values.Options{Values: req.values},
		ValuesYaml:      req.valuesYAML,
		Version:         req.chartVersion,
	},
		&helmclient.GenericHelmOptions{},
	)
	if err != nil {
		pterm.Error.Printfln("Failed to install %s Helm Chart", req.chartName)
		return fmt.Errorf("could not install helm: %w", err)
	}

	c.tel.Attr(fmt.Sprintf("helm_%s_release_version", req.name), strconv.Itoa(helmRelease.Version))

	pterm.Success.Printfln(
		"Installed Helm Chart %s:\n  Name: %s\n  Namespace: %s\n  Version: %s\n  Release: %d",
		req.chartName, helmRelease.Name, helmRelease.Namespace, helmRelease.Chart.Metadata.Version, helmRelease.Version)
	return nil
}

// openBrowser will open the url in the user's browser but only if the url returns a 200 response code first
// TODO: clean up this method, make it testable
func (c *Command) openBrowser(ctx context.Context, url string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	alive := make(chan error)

	go func() {
		tick := time.Tick(1 * time.Second)
		for {
			select {
			case <-ctx.Done():
				alive <- fmt.Errorf("liveness check failed: %w", ctx.Err())
			case <-tick:
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
				if err != nil {
					alive <- fmt.Errorf("could not create request: %w", err)
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
	case <-ctx.Done():
		pterm.Error.Println("Timed out waiting for ingress")
		return fmt.Errorf("browser liveness check failed: %w", ctx.Err())
	case err := <-alive:
		if err != nil {
			pterm.Error.Println("Ingress verification failed")
			return fmt.Errorf("browser failed liveness check: %w", err)
		}
	}
	// if we're here, then no errors occurred

	c.spinner.UpdateText(fmt.Sprintf("Attempting to launch web-browser for %s", url))

	if err := c.launcher(url); err != nil {
		pterm.Warning.Printfln("Failed to launch web-browser.\n"+
			"Please launch your web-browser to access %s", url)
		pterm.Debug.Printfln("failed to launch web-browser: %s", err.Error())
		// don't consider a failed web-browser to be a failed installation
	}

	pterm.Success.Println("Launched web-browser successfully")

	return nil
}

// defaultK8s returns the default k8s client
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

// defaultHelm returns the default helm client
func defaultHelm(kubecfg, kubectx string) (HelmClient, error) {
	k8sCfg, err := k8sClientConfig(kubecfg, kubectx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", localerr.ErrKubernetes, err)
	}

	restCfg, err := k8sCfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("%w: could not create rest config: %w", localerr.ErrKubernetes, err)
	}

	helm, err := helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
		Options:    &helmclient.Options{Namespace: airbyteNamespace, Output: &noopWriter{}, DebugLog: func(format string, v ...interface{}) {}},
		RestConfig: restCfg,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create helm client: %w", err)
	}

	return helm, nil
}

// k8sClientConfig returns a k8s client config using the ~/.kubc/config file and the k8sContext context.
func k8sClientConfig(kubecfg, kubectx string) (clientcmd.ClientConfig, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubecfg},
		&clientcmd.ConfigOverrides{CurrentContext: kubectx},
	), nil
}

// noopWriter is used by the helm client to suppress its verbose output
type noopWriter struct {
}

func (w *noopWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
