package api

import (
	"context"

	"github.com/airbytehq/abctl/internal/http"
)

// Service interface for Control Plane API operations
type Service interface {
	// Organizations
	GetOrganization(ctx context.Context, organizationID string) (*Organization, error)
	ListOrganizations(ctx context.Context) ([]*Organization, error)

	// Regions
	CreateRegion(ctx context.Context, request CreateRegionRequest) (*Region, error)
	GetRegion(ctx context.Context, regionID string) (*Region, error)
	ListRegions(ctx context.Context, organizationID string) ([]*Region, error)

	// Dataplanes
	GetDataplane(ctx context.Context, id string) (*Dataplane, error)
	ListDataplanes(ctx context.Context) ([]Dataplane, error)
	CreateDataplane(ctx context.Context, req CreateDataplaneRequest) (*CreateDataplaneResponse, error)
	DeleteDataplane(ctx context.Context, id string) error
}

// Client handles Control Plane API operations
type Client struct {
	http http.HTTPDoer
}

// NewClient creates a new API client
func NewClient(httpDoer http.HTTPDoer) *Client {
	return &Client{
		http: httpDoer,
	}
}
