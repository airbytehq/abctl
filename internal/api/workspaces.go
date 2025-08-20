package api

import (
	"context"
	"fmt"
)

// Workspace represents an Airbyte workspace
type Workspace struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name"`
	DataResidency  string `json:"dataResidency,omitempty"`
	OrganizationID string `json:"organizationId,omitempty"`
}

// CreateWorkspaceRequest represents the request to create a workspace
type CreateWorkspaceRequest struct {
	Name          string `json:"name"`
	DataResidency string `json:"dataResidency,omitempty"`
}

// UpdateWorkspaceRequest represents the request to update a workspace
type UpdateWorkspaceRequest struct {
	Name          string `json:"name,omitempty"`
	DataResidency string `json:"dataResidency,omitempty"`
}

// CreateWorkspace creates a new workspace
func (c *Client) CreateWorkspace(ctx context.Context, req *CreateWorkspaceRequest) (*Workspace, error) {
	resp, err := c.doRequest(ctx, "POST", "workspaces", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}
	
	var workspace Workspace
	if err := parseResponse(resp, &workspace); err != nil {
		return nil, err
	}
	
	return &workspace, nil
}

// UpdateWorkspace updates a workspace
func (c *Client) UpdateWorkspace(ctx context.Context, workspaceID string, req *UpdateWorkspaceRequest) (*Workspace, error) {
	endpoint := fmt.Sprintf("workspaces/%s", workspaceID)
	resp, err := c.doRequest(ctx, "PATCH", endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update workspace: %w", err)
	}
	
	var workspace Workspace
	if err := parseResponse(resp, &workspace); err != nil {
		return nil, err
	}
	
	return &workspace, nil
}

// GetWorkspace gets a workspace by ID
func (c *Client) GetWorkspace(ctx context.Context, workspaceID string) (*Workspace, error) {
	endpoint := fmt.Sprintf("workspaces/%s", workspaceID)
	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}
	
	var workspace Workspace
	if err := parseResponse(resp, &workspace); err != nil {
		return nil, err
	}
	
	return &workspace, nil
}

// ListWorkspaces lists all workspaces
func (c *Client) ListWorkspaces(ctx context.Context) ([]*Workspace, error) {
	resp, err := c.doRequest(ctx, "GET", "workspaces", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}
	
	var workspaces []*Workspace
	if err := parseResponse(resp, &workspaces); err != nil {
		return nil, err
	}
	
	return workspaces, nil
}