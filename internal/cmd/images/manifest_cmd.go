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
	}, c.ChartVersion)
	if err != nil {
		return nil, err
	}

	// Determine and set defaults for chart flags.
	err = c.setDefaultChartFlags(helmClient)
	if err != nil {
		return nil, fmt.Errorf("failed to set chart flag defaults: %w", err)
	}

	return helm.FindImagesFromChart(helmClient, valuesYaml, c.Chart, c.ChartVersion)
}

func (c *ManifestCmd) setDefaultChartFlags(helmClient goHelm.Client) error {
	resolver := helm.NewChartResolver(helmClient)
	resolvedChart, resolvedVersion, err := resolver.ResolveChartReference(c.Chart, c.ChartVersion)
	if err != nil {
		return fmt.Errorf("failed to resolve chart flags: %w", err)
	}

	c.Chart = resolvedChart
	c.ChartVersion = resolvedVersion

	return nil
}
