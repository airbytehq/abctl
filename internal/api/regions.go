package api

import (
	"context"
	"fmt"
)

// Region represents an Airbyte region
type Region struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name"`
	OrganizationID string `json:"organizationId"`
}

// CreateRegionRequest represents the request to create a region
type CreateRegionRequest struct {
	Name           string `json:"name"`
	OrganizationID string `json:"organizationId"`
}

// CreateRegion creates a new region
func (c *Client) CreateRegion(ctx context.Context, req *CreateRegionRequest) (*Region, error) {
	resp, err := c.doRequest(ctx, "POST", "regions", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create region: %w", err)
	}
	
	var region Region
	if err := parseResponse(resp, &region); err != nil {
		return nil, err
	}
	
	return &region, nil
}

// ListRegions lists all regions
func (c *Client) ListRegions(ctx context.Context, organizationID string) ([]*Region, error) {
	endpoint := fmt.Sprintf("regions?organizationId=%s", organizationID)
	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}
	
	var regions []*Region
	if err := parseResponse(resp, &regions); err != nil {
		return nil, err
	}
	
	return regions, nil
}

// GetRegion gets a region by ID
func (c *Client) GetRegion(ctx context.Context, regionID string) (*Region, error) {
	endpoint := fmt.Sprintf("regions/%s", regionID)
	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get region: %w", err)
	}
	
	var region Region
	if err := parseResponse(resp, &region); err != nil {
		return nil, err
	}
	
	return &region, nil
}