package api

import (
	"context"
	"fmt"
)

// Permission represents an Airbyte permission
type Permission struct {
	PermissionID   string `json:"permissionId"`
	PermissionType string `json:"permissionType"`
	UserID         string `json:"userId"`
	ScopeID        string `json:"scopeId"`
	Scope          string `json:"scope"`
}

// PermissionsResponse represents the API response for permissions
type PermissionsResponse struct {
	Data []*Permission `json:"data"`
}

// ListPermissions lists all permissions for the current user
func (c *Client) ListPermissions(ctx context.Context) ([]*Permission, error) {
	resp, err := c.doRequest(ctx, "GET", "permissions", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	
	var response PermissionsResponse
	if err := parseResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	
	return response.Data, nil
}

// HasPermission checks if the user has a specific permission type
func (c *Client) HasPermission(ctx context.Context, permissionType string) (bool, error) {
	permissions, err := c.ListPermissions(ctx)
	if err != nil {
		return false, err
	}
	
	for _, perm := range permissions {
		if perm.PermissionType == permissionType {
			return true, nil
		}
	}
	
	return false, nil
}

// HasInstanceAdminPermission checks if the user has instance admin permissions
func (c *Client) HasInstanceAdminPermission(ctx context.Context) (bool, error) {
	return c.HasPermission(ctx, "instance_admin")
}

// HasOrganizationAdminPermission checks if the user has organization admin permissions for a specific organization
func (c *Client) HasOrganizationAdminPermission(ctx context.Context, organizationID string) (bool, error) {
	permissions, err := c.ListPermissions(ctx)
	if err != nil {
		return false, err
	}
	
	for _, perm := range permissions {
		if perm.PermissionType == "organization_admin" && perm.ScopeID == organizationID {
			return true, nil
		}
	}
	
	return false, nil
}

// GetUserPermissions returns permissions grouped by scope and type
func (c *Client) GetUserPermissions(ctx context.Context) (map[string][]string, error) {
	permissions, err := c.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}
	
	permMap := make(map[string][]string)
	
	for _, perm := range permissions {
		key := fmt.Sprintf("%s:%s", perm.Scope, perm.ScopeID)
		permMap[key] = append(permMap[key], perm.PermissionType)
	}
	
	return permMap, nil
}

// ValidateDataPlanePermissions checks if the user has sufficient permissions to create dataplanes
func (c *Client) ValidateDataPlanePermissions(ctx context.Context, organizationID, edition string) error {
	// Check for instance admin permissions first (highest level)
	if hasInstanceAdmin, err := c.HasInstanceAdminPermission(ctx); err != nil {
		return fmt.Errorf("failed to check instance admin permissions: %w", err)
	} else if hasInstanceAdmin {
		return nil // Instance admin can create dataplanes
	}
	
	// For enterprise edition, instance admin is required
	if edition == "enterprise" {
		return fmt.Errorf("insufficient permissions: enterprise edition requires 'instance_admin' permissions to create dataplanes")
	}
	
	// For cloud/oss editions, check organization admin permissions
	if hasOrgAdmin, err := c.HasOrganizationAdminPermission(ctx, organizationID); err != nil {
		return fmt.Errorf("failed to check organization admin permissions: %w", err)
	} else if hasOrgAdmin {
		return nil // Organization admin can create dataplanes for their org
	}
	
	// Get all permissions for detailed error message
	permissions, err := c.ListPermissions(ctx)
	if err != nil {
		return fmt.Errorf("insufficient permissions to create dataplanes (failed to retrieve permissions: %w)", err)
	}
	
	// Build error message with current permissions
	var permTypes []string
	for _, perm := range permissions {
		permTypes = append(permTypes, fmt.Sprintf("%s (%s: %s)", perm.PermissionType, perm.Scope, perm.ScopeID))
	}
	
	if len(permTypes) == 0 {
		return fmt.Errorf("insufficient permissions: no permissions found. You need 'instance_admin' or 'organization_admin' permissions to create dataplanes")
	}
	
	return fmt.Errorf("insufficient permissions to create dataplanes. Required: 'instance_admin' or 'organization_admin' for organization %s. Current permissions: %v", organizationID, permTypes)
}