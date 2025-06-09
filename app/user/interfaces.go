package user

import (
	"context"

	"github.com/google/uuid"

	"github.com/joefazee/neo/models"
)

type Repository interface {
	Create(ctx context.Context, user *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByPhone(ctx context.Context, phone string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	AssignRole(ctx context.Context, userID, roleID uuid.UUID) error
	GetByIDWithPermissions(ctx context.Context, userID uuid.UUID) (*models.User, error)

	GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
	GetUsers(ctx context.Context, filters *AdminUserFilters) ([]models.User, int64, error)
	UpdateUserStatus(ctx context.Context, userID uuid.UUID, isActive bool) error
	BulkAssignPermissions(ctx context.Context, userIDs, permissionIDs []uuid.UUID) error
}

type Service interface {
	Register(ctx context.Context, req *RegisterUserRequest) (*Response, error)
	Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error)
	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	AssignRole(ctx context.Context, userID, roleID uuid.UUID) error
}
