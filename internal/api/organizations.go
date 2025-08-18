package api

import (
	"context"
	"fmt"
)

// Organization represents an Airbyte organization
type Organization struct {
	ID    string `json:"organizationId"`
	Name  string `json:"organizationName"`
	Email string `json:"email"`
}

// OrganizationResponse represents the API response for organizations
type OrganizationResponse struct {
	Data []*Organization `json:"data"`
}

// ListOrganizations lists all organizations the user has access to
func (c *Client) ListOrganizations(ctx context.Context) ([]*Organization, error) {
	resp, err := c.doRequest(ctx, "GET", "organizations", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	
	var response OrganizationResponse
	if err := parseResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	
	return response.Data, nil
}

// GetOrganization gets an organization by ID
func (c *Client) GetOrganization(ctx context.Context, organizationID string) (*Organization, error) {
	endpoint := fmt.Sprintf("organizations/%s", organizationID)
	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	
	var org Organization
	if err := parseResponse(resp, &org); err != nil {
		return nil, err
	}
	
	return &org, nil
}