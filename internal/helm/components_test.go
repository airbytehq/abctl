package helm

import (
	"context"
	"testing"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/api"
	"github.com/airbytehq/abctl/internal/helm/mock"
	goHelm "github.com/mittwald/go-helm-client"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/release"
)

func TestComponentOverlayMerge(t *testing.T) {
	base := map[string]any{"nested": map[string]any{"keep": true, "replace": 1}, "list": []any{1, 2}}
	overlay := map[string]any{"nested": map[string]any{"replace": 2}, "list": []any{}, "nil": nil}
	merged := MergeValues(base, overlay)
	require.Equal(t, map[string]any{"keep": true, "replace": 2}, merged["nested"])
	require.Empty(t, merged["list"])
	require.Nil(t, merged["nil"])
	require.Equal(t, 1, base["nested"].(map[string]any)["replace"])
	require.Equal(t, map[string]any{}, MergeValues(nil, nil))
}

func TestComponentHelmOptions(t *testing.T) {
	for _, deferWait := range []bool{false, true} {
		client := mock.NewMockClient(gomock.NewController(t))
		client.EXPECT().AddOrUpdateChartRepo(gomock.Any()).Return(nil)
		client.EXPECT().InstallOrUpgradeChart(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, spec *goHelm.ChartSpec, _ *goHelm.GenericHelmOptions) (*release.Release, error) {
			require.Equal(t, !deferWait, spec.Wait)
			require.Equal(t, !deferWait, spec.Atomic)
			var values map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(spec.ValuesYaml), &values))
			require.Equal(t, "primary", values["targetCluster"])
			require.Equal(t, "real-dp", values["dataPlane"].(map[string]any)["id"])
			require.Equal(t, "gcs", values["storage"].(map[string]any)["type"])
			require.Equal(t, "custom-agents", values["componentsNamespace"])
			return &release.Release{}, nil
		})
		err := InstallDataplaneChart(context.Background(), client, "ns", "release", "chart", "2.1.0", "repo", &api.CreateDataplaneResponse{DataplaneID: "real-dp", ClientID: "id", ClientSecret: "test-only"}, &airbox.Context{Edition: "cloud"}, DataplaneChartOptions{TargetCluster: "primary", DeferWait: deferWait, Values: map[string]any{"targetCluster": "wrong", "dataPlane": map[string]any{"id": "wrong"}, "storage": map[string]any{"type": "gcs"}, "componentsNamespace": "custom-agents"}})
		require.NoError(t, err)
	}
}
