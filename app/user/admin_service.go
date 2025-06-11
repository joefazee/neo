package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/joefazee/neo/models"
	"gorm.io/gorm"

	"github.com/google/uuid"
)

type AdminService interface {
	GetUsers(ctx context.Context, filters *AdminUserFilters) ([]AdminUserResponse, int64, error)
	UpdateUserStatus(ctx context.Context, userID uuid.UUID, isActive bool) error
	AssignRole(ctx context.Context, userID, roleID uuid.UUID) error
	BulkAssignPermissions(ctx context.Context, userIDs, permissionIDs []uuid.UUID) error

	CreatePermission(ctx context.Context, req *CreatePermissionRequest) (*PermissionResponse, error)

	CreateRole(ctx context.Context, req *CreateRoleRequest) (*RoleResponse, error)
	UpdateRole(ctx context.Context, id uuid.UUID, req *UpdateRoleRequest) (*RoleResponse, error)
	AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, req *AssignPermissionsRequest) (*RoleResponse, error)
	RemovePermissionsFromRole(ctx context.Context, roleID uuid.UUID, req *RemovePermissionsRequest) (*RoleResponse, error)

	GetUserByID(ctx context.Context, id uuid.UUID) (*AdminUserResponse, error)
	RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) (*AdminUserResponse, error)
}

type adminService struct {
	repo Repository
}

func NewAdminService(repo Repository) AdminService {
	return &adminService{repo: repo}
}

func (s *adminService) GetUsers(ctx context.Context, filters *AdminUserFilters) ([]AdminUserResponse, int64, error) {
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PerPage < 1 || filters.PerPage > 100 {
		filters.PerPage = 20
	}

	users, total, err := s.repo.GetUsers(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	userResponses := make([]AdminUserResponse, 0, len(users))
	for i := range users {
		user := users[i]
		var roles []*RoleResponse
		for _, role := range user.Roles {
			roles = append(roles, ToRoleResponse(&role))
		}
		userResponses = append(userResponses, AdminUserResponse{
			ID:        user.ID,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
			Phone:     user.Phone,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
			Roles:     roles,
		})
	}

	return userResponses, total, nil
}

func (s *adminService) UpdateUserStatus(ctx context.Context, userID uuid.UUID, isActive bool) error {
	return s.repo.UpdateUserStatus(ctx, userID, isActive)
}

func (s *adminService) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return s.repo.AssignRole(ctx, userID, roleID)
}

func (s *adminService) BulkAssignPermissions(ctx context.Context, userIDs, permissionIDs []uuid.UUID) error {
	return s.repo.BulkAssignPermissions(ctx, userIDs, permissionIDs)
}

func (s *adminService) CreatePermission(ctx context.Context, req *CreatePermissionRequest) (*PermissionResponse, error) {
	// Check if permission already exists
	existing, err := s.repo.GetPermissionByName(ctx, req.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing permission: %w", err)
	}
	if existing != nil {
		return nil, errors.New("permission with this name already exists")
	}

	permission := &models.Permission{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.repo.CreatePermission(ctx, permission); err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	return ToPermissionResponse(permission), nil
}

func (s *adminService) CreateRole(ctx context.Context, req *CreateRoleRequest) (*RoleResponse, error) {
	role := &models.Role{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.repo.CreateRole(ctx, role); err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return ToRoleResponse(role), nil
}

func (s *adminService) UpdateRole(ctx context.Context, id uuid.UUID, req *UpdateRoleRequest) (*RoleResponse, error) {
	role, err := s.repo.GetRoleByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// Update fields if provided
	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Description != nil {
		role.Description = *req.Description
	}

	if err := s.repo.UpdateRole(ctx, role); err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	return ToRoleResponse(role), nil
}

// AssignPermissionsToRole assigns permissions to a role by their codes.
// //nolint: dupl
func (s *adminService) AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, req *AssignPermissionsRequest) (*RoleResponse, error) {
	// Get permissions by their codes
	permissions, err := s.repo.GetPermissionsByNames(ctx, req.PermissionCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}

	if len(permissions) != len(req.PermissionCodes) {
		return nil, errors.New("one or more permission codes not found")
	}

	// Extract permission IDs
	permissionIDs := make([]uuid.UUID, len(permissions))
	for i, perm := range permissions {
		permissionIDs[i] = perm.ID
	}

	// Assign permissions to role
	if err := s.repo.AssignPermissionsToRole(ctx, roleID, permissionIDs); err != nil {
		return nil, fmt.Errorf("failed to assign permissions to role: %w", err)
	}

	// Get updated role with permissions
	updatedRole, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated role: %w", err)
	}

	return ToRoleResponse(updatedRole), nil
}

// RemovePermissionsFromRole removes permissions from a role by their codes.
// //nolint: dupl
func (s *adminService) RemovePermissionsFromRole(ctx context.Context, roleID uuid.UUID, req *RemovePermissionsRequest) (*RoleResponse, error) {
	// Get permissions by their codes
	permissions, err := s.repo.GetPermissionsByNames(ctx, req.PermissionCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}

	if len(permissions) != len(req.PermissionCodes) {
		return nil, errors.New("one or more permission codes not found")
	}

	// Extract permission IDs
	permissionIDs := make([]uuid.UUID, len(permissions))
	for i, perm := range permissions {
		permissionIDs[i] = perm.ID
	}

	// Remove permissions from role
	if err := s.repo.RemovePermissionsFromRole(ctx, roleID, permissionIDs); err != nil {
		return nil, fmt.Errorf("failed to remove permissions from role: %w", err)
	}

	// Get updated role with permissions
	updatedRole, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated role: %w", err)
	}

	return ToRoleResponse(updatedRole), nil
}

func (s *adminService) GetUserByID(ctx context.Context, id uuid.UUID) (*AdminUserResponse, error) {
	user, err := s.repo.GetUserByIDWithRoles(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return ToUserResponse(user), nil
}

func (s *adminService) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) (*AdminUserResponse, error) {
	// Check if user exists
	user, err := s.repo.GetUserByIDWithRoles(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user has the role
	hasRole := false
	for _, role := range user.Roles {
		if role.ID == roleID {
			hasRole = true
			break
		}
	}

	if !hasRole {
		return nil, errors.New("user does not have this role")
	}

	// Remove role from user
	if err := s.repo.RemoveRoleFromUser(ctx, userID, roleID); err != nil {
		return nil, fmt.Errorf("failed to remove role from user: %w", err)
	}

	// Get updated user
	updatedUser, err := s.repo.GetUserByIDWithRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated user: %w", err)
	}

	return ToUserResponse(updatedUser), nil
}
