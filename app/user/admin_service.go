package user

import (
	"context"

	"github.com/google/uuid"
)

type AdminService interface {
	GetUsers(ctx context.Context, filters *AdminUserFilters) ([]AdminUserResponse, int64, error)
	UpdateUserStatus(ctx context.Context, userID uuid.UUID, isActive bool) error
	AssignRole(ctx context.Context, userID, roleID uuid.UUID) error
	BulkAssignPermissions(ctx context.Context, userIDs, permissionIDs []uuid.UUID) error
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
		var roleNames []string
		for _, role := range user.Roles {
			roleNames = append(roleNames, role.Name)
		}
		userResponses = append(userResponses, AdminUserResponse{
			ID:        user.ID,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
			Phone:     user.Phone,
			IsActive:  *user.IsActive,
			CreatedAt: user.CreatedAt,
			Roles:     roleNames,
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
