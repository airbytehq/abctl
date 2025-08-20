package api

import (
	"context"
	"fmt"
	"net/url"
)

// Region represents an Airbyte region
type Region struct {
	ID             string `json:"regionId,omitempty"`
	Name           string `json:"name"`
	OrganizationID string `json:"organizationId"`
}

// CreateRegionRequest represents the request to create a region
type CreateRegionRequest struct {
	Name           string `json:"name"`
	OrganizationID string `json:"organizationId"`
}

// CreateRegionResponse represents the response from creating a region
type CreateRegionResponse struct {
	ID             string `json:"regionId"`
	Name           string `json:"name"`
	OrganizationID string `json:"organizationId"`
}

// CreateRegion creates a new region
func (c *Client) CreateRegion(ctx context.Context, req *CreateRegionRequest) (*Region, error) {
	resp, err := c.doRequest(ctx, "POST", "regions", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create region: %w", err)
	}
	
	var response CreateRegionResponse
	if err := parseResponse(resp, &response); err != nil {
		return nil, err
	}
	
	// Convert response to Region struct
	region := &Region{
		ID:             response.ID,
		Name:           response.Name,
		OrganizationID: response.OrganizationID,
	}
	
	return region, nil
}

// ListRegions lists regions, optionally filtered by organization
func (c *Client) ListRegions(ctx context.Context, organizationID string) ([]*Region, error) {
	endpoint := "regions"
	var queryParams url.Values
	
	if organizationID != "" {
		queryParams = url.Values{}
		queryParams.Set("organizationId", organizationID)
	}
	
	resp, err := c.doRequestWithQuery(ctx, "GET", endpoint, queryParams, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}
	
	var regions []*Region
	if err := parseResponse(resp, &regions); err != nil {
		return nil, err
	}
	
	return regions, nil
}

// ListAllRegions lists all regions without filtering
func (c *Client) ListAllRegions(ctx context.Context) ([]*Region, error) {
	return c.ListRegions(ctx, "")
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