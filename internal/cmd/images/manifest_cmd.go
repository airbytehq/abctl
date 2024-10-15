package images

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/airbytehq/abctl/internal/cmd/local/helm"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/trace"
	helmlib "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/airbytehq/abctl/internal/common"
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

func (c *ManifestCmd) Run(ctx context.Context, provider k8s.Provider) error {
	ctx, span := trace.NewSpan(ctx, "images manifest")
	defer span.End()

	client, err := helm.New(provider.Kubeconfig, provider.Context, common.AirbyteNamespace)
	if err != nil {
		return err
	}

	images, err := c.findAirbyteImages(ctx, client)
	if err != nil {
		return err
	}

	for _, img := range images {
		fmt.Println(img)
	}

	return nil
}

func (c *ManifestCmd) findAirbyteImages(ctx context.Context, client helm.Client) ([]string, error) {
	valuesYaml, err := helm.BuildAirbyteValues(ctx, helm.ValuesOpts{
		ValuesFile: c.Values,
	})
	if err != nil {
		return nil, err
	}

	airbyteChartLoc := helm.LocateLatestAirbyteChart(c.ChartVersion, c.Chart)
	return findImagesFromChart(client, valuesYaml, airbyteChartLoc, c.ChartVersion)
}

func findImagesFromChart(client helm.Client, valuesYaml, chartName, chartVersion string) ([]string, error) {
	err := client.AddOrUpdateChartRepo(repo.Entry{
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
	imageSet := set[string]{}

	for _, obj := range objs {

		var podSpec *corev1.PodSpec
		switch z := obj.(type) {
		case *corev1.ConfigMap:
			if strings.HasSuffix(z.Name, "airbyte-env") {
				for k, v := range z.Data {
					if strings.HasSuffix(k, "_IMAGE") {
						imageSet.add(v)
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
			imageSet.add(c.Image)
		}
		for _, c := range podSpec.Containers {
			imageSet.add(c.Image)
		}
	}

	var out []string
	for _, k := range imageSet.items() {
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

type set[T comparable] struct {
	vals map[T]struct{}
}

func (s *set[T]) add(v T) {
	if s.vals == nil {
		s.vals = map[T]struct{}{}
	}
	s.vals[v] = struct{}{}
}

func (s *set[T]) items() []T {
	out := make([]T, len(s.vals))
	for k := range s.vals {
		out = append(out, k)
	}
	return out
}
