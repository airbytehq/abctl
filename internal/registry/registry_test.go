package registry

import (
	"context"
	"github.com/airbytehq/abctl/internal/cmd/local/docker"
	"testing"
)

func TestRegistry_Start(t *testing.T) {
	//d, _ := docker.Docker{Client: dockertest.MockClient{
	//	FnImageList: func(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	//		return []image.Summary{{ID: name}}, nil
	//	},
	//}}
	ctx := context.Background()
	d, _ := docker.New(ctx)
	r := &Registry{d: d}

	r.Start(ctx)
}
