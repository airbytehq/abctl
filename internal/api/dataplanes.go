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
	dataplanesPath = "/v1/dataplanes"
)

// UpdateDataplaneRequest represents the input for dataplane update operations
type UpdateDataplaneRequest struct {
	Name     string `json:"name"`
	RegionID string `json:"regionId"`
	Enabled  bool   `json:"enabled"`
}

// CreateDataplaneRequest represents the input for creating a new dataplane
type CreateDataplaneRequest struct {
	Name           string `json:"name"`
	RegionID       string `json:"regionId"`
	OrganizationID string `json:"organizationId"`
	Enabled        bool   `json:"enabled"`
}

// Dataplane represents a dataplane resource for GET/LIST operations
type Dataplane struct {
	DataplaneID string `json:"dataplaneId" yaml:"dataplaneId"`
	Name        string `json:"name" yaml:"name"`
	RegionID    string `json:"regionId" yaml:"regionId"`
	Enabled     bool   `json:"enabled" yaml:"enabled"`
}

// CreateDataplaneResponse represents the response from creating a dataplane
type CreateDataplaneResponse struct {
	DataplaneID  string `json:"dataplaneId"`
	RegionID     string `json:"regionId"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

// GetDataplane retrieves a specific dataplane by ID
func (c *Client) GetDataplane(ctx context.Context, id string) (*Dataplane, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dataplanesPath+"/"+id, nil)
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

	var dataplane Dataplane
	if err := json.NewDecoder(resp.Body).Decode(&dataplane); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &dataplane, nil
}

// ListDataplanes retrieves all dataplanes
func (c *Client) ListDataplanes(ctx context.Context) ([]Dataplane, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dataplanesPath, nil)
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

	var dataplanes []Dataplane
	if err := json.NewDecoder(resp.Body).Decode(&dataplanes); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return dataplanes, nil
}

// CreateDataplane creates a new dataplane
func (c *Client) CreateDataplane(ctx context.Context, req CreateDataplaneRequest) (*CreateDataplaneResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, dataplanesPath, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() // Connection cleanup, error doesn't affect functionality

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var response CreateDataplaneResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// DeleteDataplane deletes a dataplane by ID
func (c *Client) DeleteDataplane(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, dataplanesPath+"/"+id, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() // Connection cleanup, error doesn't affect functionality

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
