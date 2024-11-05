package images

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/airbytehq/abctl/internal/cmd/local/helm"
	"github.com/airbytehq/abctl/internal/common"
	"github.com/airbytehq/abctl/internal/trace"
	helmlib "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/repo"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/scheme"
)

type ManifestCmd struct {
	Chart        string `help:"Path to chart." xor:"chartver"`
	ChartVersion string `help:"Version of the chart." xor:"chartver"`
	Values       string `type:"existingfile" help:"An Airbyte helm chart values file to configure helm."`
}

func (c *ManifestCmd) Run(ctx context.Context) error {
	ctx, span := trace.NewSpan(ctx, "images manifest")
	defer span.End()

	images, err := c.findAirbyteImages(ctx)
	if err != nil {
		return err
	}

	for _, img := range images {
		fmt.Println(img)
	}

	return nil
}

func (c *ManifestCmd) findAirbyteImages(ctx context.Context) ([]string, error) {
	valuesYaml, err := helm.BuildAirbyteValues(ctx, helm.ValuesOpts{
		ValuesFile: c.Values,
	})
	if err != nil {
		return nil, err
	}

	airbyteChartLoc := helm.LocateLatestAirbyteChart(c.ChartVersion, c.Chart)
	return FindImagesFromChart(valuesYaml, airbyteChartLoc, c.ChartVersion)
}

func FindImagesFromChart(valuesYaml, chartName, chartVersion string) ([]string, error) {

	// sharing a helm client with the install code causes some weird issues,
	// and templating the chart doesn't need details about the k8s provider,
	// we create a throwaway helm client here.
	client, err := helmlib.New(helm.ClientOptions(common.AirbyteNamespace))
	if err != nil {
		return nil, err
	}

	err = client.AddOrUpdateChartRepo(repo.Entry{
		Name: common.AirbyteRepoName,
		URL:  common.AirbyteRepoURL,
	})
	if err != nil {
		return nil, err
	}

	bytes, err := client.TemplateChart(&helmlib.ChartSpec{
		ChartName:    chartName,
		GenerateName: true,
		ValuesYaml:   valuesYaml,
		Version:      chartVersion,
	}, nil)
	if err != nil {
		return nil, err
	}

	images := findAllImages(string(bytes))
	return images, nil
}

// findAllImages walks through the Helm chart, looking for container images in k8s PodSpecs.
// It also looks for env vars in the airbyte-env config map that end with "_IMAGE".
// It returns a unique, sorted list of images found.
func findAllImages(chartYaml string) []string {
	objs := decodeK8sResources(chartYaml)
	imageSet := common.Set[string]{}

	for _, obj := range objs {

		var podSpec *corev1.PodSpec
		switch z := obj.(type) {
		case *corev1.ConfigMap:
			if strings.HasSuffix(z.Name, "airbyte-env") {
				for k, v := range z.Data {
					if strings.HasSuffix(k, "_IMAGE") {
						imageSet.Add(v)
					}
				}
			}
			continue
		case *corev1.Pod:
			podSpec = &z.Spec
		case *batchv1.Job:
			podSpec = &z.Spec.Template.Spec
		case *appsv1.Deployment:
			podSpec = &z.Spec.Template.Spec
		case *appsv1.StatefulSet:
			podSpec = &z.Spec.Template.Spec
		default:
			continue
		}

		for _, c := range podSpec.InitContainers {
			imageSet.Add(c.Image)
		}
		for _, c := range podSpec.Containers {
			imageSet.Add(c.Image)
		}
	}

	var out []string
	for _, k := range imageSet.Items() {
		if k != "" {
			out = append(out, k)
		}
	}
	slices.Sort(out)

	return out
}

func decodeK8sResources(renderedYaml string) []runtime.Object {
	out := []runtime.Object{}
	chunks := strings.Split(renderedYaml, "---")
	for _, chunk := range chunks {
		if len(chunk) == 0 {
			continue
		}
		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(chunk), nil, nil)
		if err != nil {
			continue
		}
		out = append(out, obj)
	}
	return out
}
