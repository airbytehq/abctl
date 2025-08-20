package api

import (
	"context"
	"fmt"
)

// InstanceConfiguration represents the instance configuration
type InstanceConfiguration struct {
	Edition          string `json:"edition"`
	Version          string `json:"version"`
	LicenseType      string `json:"licenseType,omitempty"`
	TrackingStrategy string `json:"trackingStrategy,omitempty"`
}

// GetInstanceConfiguration gets the instance configuration
func (c *Client) GetInstanceConfiguration(ctx context.Context) (*InstanceConfiguration, error) {
	// Note: This endpoint uses v1 instead of public/v1
	resp, err := c.doRequestWithPath(ctx, "GET", "/api/v1/instance_configuration", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance configuration: %w", err)
	}
	
	var config InstanceConfiguration
	if err := parseResponse(resp, &config); err != nil {
		return nil, err
	}
	
	return &config, nil
}