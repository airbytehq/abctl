package migrate

import (
	"context"
	"errors"
	"github.com/airbytehq/abctl/internal/cmd/local/docker/dockertest"
	"github.com/docker/docker/api/types/volume"
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestFromDockerVolume(t *testing.T) {

}

func Test_volumeExists(t *testing.T) {
	tests := []struct {
		name  string
		vol   volume.Volume
		volID string
		err   error
		exp   string
	}{
		{
			name:  "missing volume",
			volID: "volume",
			err:   errors.New("error"),
			exp:   "",
		},
		{
			name:  "existing volume",
			vol:   volume.Volume{Mountpoint: "mountpoint"},
			volID: "volume",
			exp:   "mountpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := dockertest.MockClient{
				FnVolumeInspect: func(ctx context.Context, volumeID string) (volume.Volume, error) {
					if d := cmp.Diff(tt.volID, volumeID); d != "" {
						t.Errorf("volume mismatch (-want +got): %s", d)
					}
					return tt.vol, tt.err
				},
			}

			if d := cmp.Diff(tt.exp, volumeExists(context.Background(), cli, tt.volID)); d != "" {
				t.Errorf("mismatch (-want +got): %s", d)
			}
		})
	}
}
