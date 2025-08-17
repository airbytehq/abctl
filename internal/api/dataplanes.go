package api

import (
	"context"
	"fmt"
)

// DataPlane represents an Airbyte data plane
type DataPlane struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	RegionID string `json:"regionId"`
	Status   string `json:"status,omitempty"`
}

// CreateDataPlaneRequest represents the request to create a data plane
type CreateDataPlaneRequest struct {
	Name     string `json:"name"`
	RegionID string `json:"regionId"`
}

// CreateDataPlane creates a new data plane
func (c *Client) CreateDataPlane(ctx context.Context, req *CreateDataPlaneRequest) (*DataPlane, error) {
	resp, err := c.doRequest(ctx, "POST", "dataplanes", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create data plane: %w", err)
	}
	
	var dataPlane DataPlane
	if err := parseResponse(resp, &dataPlane); err != nil {
		return nil, err
	}
	
	return &dataPlane, nil
}

// ListDataPlanes lists all data planes
func (c *Client) ListDataPlanes(ctx context.Context, regionID string) ([]*DataPlane, error) {
	endpoint := "dataplanes"
	if regionID != "" {
		endpoint = fmt.Sprintf("dataplanes?regionId=%s", regionID)
	}
	
	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list data planes: %w", err)
	}
	
	var dataPlanes []*DataPlane
	if err := parseResponse(resp, &dataPlanes); err != nil {
		return nil, err
	}
	
	return dataPlanes, nil
}

// GetDataPlane gets a data plane by ID
func (c *Client) GetDataPlane(ctx context.Context, dataPlaneID string) (*DataPlane, error) {
	endpoint := fmt.Sprintf("dataplanes/%s", dataPlaneID)
	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get data plane: %w", err)
	}
	
	var dataPlane DataPlane
	if err := parseResponse(resp, &dataPlane); err != nil {
		return nil, err
	}
	
	return &dataPlane, nil
}

// DeleteDataPlane deletes a data plane
func (c *Client) DeleteDataPlane(ctx context.Context, dataPlaneID string) error {
	endpoint := fmt.Sprintf("dataplanes/%s", dataPlaneID)
	resp, err := c.doRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to delete data plane: %w", err)
	}
	
	return parseResponse(resp, nil)
}