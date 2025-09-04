package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	organizationsPath = "/v1/organizations"
)

// Organization represents an Airbyte organization
type Organization struct {
	ID    string `json:"organizationId"`
	Name  string `json:"organizationName"`
	Email string `json:"email"`
}

// Pagination represents pagination parameters
type Pagination struct {
	PageSize  int `json:"pageSize,omitempty"`
	RowOffset int `json:"rowOffset,omitempty"`
}

// ListOrganizationsOptions holds query options for listing organizations
type ListOrganizationsOptions struct {
	UserID       string
	NameContains string
	Pagination   *Pagination
}

// ListOrganizationsOption is a functional option for configuring organization queries
type ListOrganizationsOption func(*ListOrganizationsOptions)

// WithNameContains filters organizations by name
func WithNameContains(name string) ListOrganizationsOption {
	return func(opts *ListOrganizationsOptions) {
		opts.NameContains = name
	}
}

// WithPagination sets pagination parameters
func WithPagination(pagination *Pagination) ListOrganizationsOption {
	return func(opts *ListOrganizationsOptions) {
		opts.Pagination = pagination
	}
}

// GetOrganization retrieves a specific organization by ID
func (c *Client) GetOrganization(ctx context.Context, organizationID string) (*Organization, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, organizationsPath+"/"+organizationID, nil)
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

	var organization Organization
	if err := json.NewDecoder(resp.Body).Decode(&organization); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &organization, nil
}

// ListOrganizations retrieves organizations for the authenticated user
func (c *Client) ListOrganizations(ctx context.Context) ([]*Organization, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, organizationsPath, nil)
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

	// Decode response according to OpenAPI spec: OrganizationsResponse with data field
	var response struct {
		Data []*Organization `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Data, nil
}
