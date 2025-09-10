package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	regionsPath = "/v1/regions"
)

// Region represents an Airbyte region
type Region struct {
	ID            string `json:"regionId"`
	Name          string `json:"name"`
	CloudProvider string `json:"cloudProvider,omitempty"`
	Location      string `json:"location,omitempty"`
	Status        string `json:"status,omitempty"`
}

// RegionsResponse represents the response from listing regions
type RegionsResponse struct {
	Regions []Region `json:"regions"`
}

// CreateRegionRequest represents the request to create a new region
type CreateRegionRequest struct {
	Name           string `json:"name"`
	OrganizationID string `json:"organizationId"`
	CloudProvider  string `json:"cloudProvider,omitempty"`
	Location       string `json:"location,omitempty"`
}

// CreateRegion creates a new region
func (c *Client) CreateRegion(ctx context.Context, request CreateRegionRequest) (*Region, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, regionsPath, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() // Connection cleanup, error doesn't affect functionality

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var region Region
	if err := json.Unmarshal(body, &region); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &region, nil
}

// GetRegion retrieves a specific region by ID
func (c *Client) GetRegion(ctx context.Context, regionID string) (*Region, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, regionsPath+"/"+regionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() // Connection cleanup, error doesn't affect functionality

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var region Region
	if err := json.NewDecoder(resp.Body).Decode(&region); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &region, nil
}

// ListRegions retrieves all regions for an organization
func (c *Client) ListRegions(ctx context.Context, organizationID string) ([]*Region, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, regionsPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if organizationID != "" {
		q := req.URL.Query()
		q.Set("organizationId", organizationID)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() // Connection cleanup, error doesn't affect functionality

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var regions []*Region
	if err := json.NewDecoder(resp.Body).Decode(&regions); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return regions, nil
}
