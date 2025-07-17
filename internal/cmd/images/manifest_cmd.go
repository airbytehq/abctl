package images

import (
	"context"
	"fmt"

	goHelm "github.com/mittwald/go-helm-client"

	"github.com/airbytehq/abctl/internal/common"
	"github.com/airbytehq/abctl/internal/helm"
	"github.com/airbytehq/abctl/internal/paths"
	"github.com/airbytehq/abctl/internal/service"
	"github.com/airbytehq/abctl/internal/trace"
)

type ManifestCmd struct {
	Chart        string `help:"Path to chart." xor:"chartver"`
	ChartVersion string `help:"Version of the chart." xor:"chartver"`
	Values       string `type:"existingfile" help:"An Airbyte helm chart values file to configure helm."`
}

func (c *ManifestCmd) Run(ctx context.Context, newSvcMgrClients service.ManagerClientFactory) error {
	ctx, span := trace.NewSpan(ctx, "images manifest")
	defer span.End()

	// Load the required service manager clients. We only need the Helm client
	// for image manifest operations.
	_, helmClient, err := newSvcMgrClients(paths.Kubeconfig, common.AirbyteKubeContext)
	if err != nil {
		return err
	}

	images, err := c.findAirbyteImages(ctx, helmClient)
	if err != nil {
		return err
	}

	for _, img := range images {
		fmt.Println(img)
	}

	return nil
}

func (c *ManifestCmd) findAirbyteImages(ctx context.Context, helmClient goHelm.Client) ([]string, error) {
	valuesYaml, err := helm.BuildAirbyteValues(ctx, helm.ValuesOpts{
		ValuesFile: c.Values,
	})
	if err != nil {
		return nil, err
	}

	airbyteChartLoc := helm.LocateLatestAirbyteChart(c.ChartVersion, c.Chart)
	return helm.FindImagesFromChart(helmClient, valuesYaml, airbyteChartLoc, c.ChartVersion)
}
