package migrate

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/airbytehq/abctl/internal/cmd/local/docker/dockertest"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/volume"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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

func Test_ensureImage(t *testing.T) {
	errTest := errors.New("error")
	tests := []struct {
		name           string
		img            string
		imgListSummary []image.Summary
		imgListErr     error
		imgPull        io.ReadCloser
		imgPullErr     error
		exp            error
	}{
		{
			name:           "image already exists",
			img:            "image",
			imgListSummary: []image.Summary{{ID: "imageID"}},
		},
		{
			name:       "error listing images",
			img:        "image",
			imgListErr: errTest,
			exp:        errTest, // the actual error will wrap this one
		},
		{
			name:       "error pulling image",
			img:        "image",
			imgPullErr: errTest,
			exp:        errTest,
		},
		{
			name:    "successfully pulled image",
			img:     "image-success",
			imgPull: io.NopCloser(strings.NewReader("image-success")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := dockertest.MockClient{
				FnImageList: func(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
					if d := cmp.Diff([]string{tt.img}, options.Filters.Get("reference")); d != "" {
						t.Errorf("image filter mismatch (-want +got): %s", d)
					}

					return tt.imgListSummary, tt.imgListErr
				},

				FnImagePull: func(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error) {
					if d := cmp.Diff(tt.img, refStr); d != "" {
						t.Errorf("image mismatch (-want +got): %s", d)
					}

					return tt.imgPull, tt.imgPullErr
				},
			}

			err := ensureImage(context.Background(), cli, tt.img)
			if d := cmp.Diff(tt.exp, err, cmpopts.EquateErrors()); d != "" {
				t.Errorf("error mismatch (-want +got): %s", d)
			}
		})
	}
}
