package user

import (
	"context"

	"github.com/joefazee/neo/models"
)

type Repository interface {
	Create(ctx context.Context, user *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByPhone(ctx context.Context, phone string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
}

type Service interface {
	Register(ctx context.Context, req *RegisterUserRequest) (*Response, error)
	Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error)
	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
}
