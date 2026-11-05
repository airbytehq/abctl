package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type RegisterComponentRequest struct {
	DataplaneID string `json:"dataplaneId"`
	Component   string `json:"component"`
	URL         string `json:"url"`
}

type UnregisterComponentRequest struct {
	DataplaneID string `json:"dataplaneId"`
	Component   string `json:"component"`
}

type ComponentCredentials struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

func (c *Client) RegisterComponent(ctx context.Context, req RegisterComponentRequest) (*ComponentCredentials, error) {
	var credentials ComponentCredentials
	if err := c.componentRequest(ctx, "register_component", req, http.StatusOK, &credentials); err != nil {
		return nil, err
	}
	return &credentials, nil
}

func (c *Client) UnregisterComponent(ctx context.Context, req UnregisterComponentRequest) error {
	return c.componentRequest(ctx, "unregister_component", req, http.StatusNoContent, nil)
}

func (c *Client) componentRequest(ctx context.Context, operation string, input any, status int, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("component request encoding failed: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/dataplanes/"+operation, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("component request creation failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("component %s request failed: %w", operation, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != status {
		// Registration responses may contain credentials; never echo response bodies.
		return fmt.Errorf("component %s returned HTTP %d", operation, resp.StatusCode)
	}
	if output != nil {
		if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
			return fmt.Errorf("component %s response is not valid JSON", operation)
		}
	}
	return nil
}
