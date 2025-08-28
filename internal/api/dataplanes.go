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
	dataplanesPath = "/api/v1/dataplanes"
)

// DataplaneSpec represents the input for dataplane operations
type DataplaneSpec struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Config map[string]string `json:"config"`
}

// Dataplane represents a dataplane resource
type Dataplane struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Status string            `json:"status"`
	Config map[string]string `json:"config"`
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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
func (c *Client) CreateDataplane(ctx context.Context, spec DataplaneSpec) (*Dataplane, error) {
	reqBody, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dataplanesPath, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var dataplane Dataplane
	if err := json.NewDecoder(resp.Body).Decode(&dataplane); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &dataplane, nil
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
