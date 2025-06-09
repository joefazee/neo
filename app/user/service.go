package user

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/joefazee/neo/internal/security"
	"github.com/joefazee/neo/models"
	"gorm.io/gorm"
)

type service struct {
	repo       Repository
	tokenMaker security.Maker
}

// NewService creates a new user service.
func NewService(repo Repository, tokenMaker security.Maker) Service {
	return &service{
		repo:       repo,
		tokenMaker: tokenMaker,
	}
}

func (s *service) Register(ctx context.Context, req *RegisterUserRequest) (*Response, error) {
	hashedPassword, err := models.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        req.Email,
		Phone:        req.PhoneNumber,
		PasswordHash: hashedPassword,
		CountryID:    uuid.MustParse("65a8ad97-b196-4b32-b82b-9d3bc6b74e95"),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return &Response{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Phone:     user.Phone,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (s *service) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	var user *models.User
	var err error

	// Determine if the identity is an email or phone number
	if models.IsEmail(req.Identity) {
		user, err = s.repo.GetByEmail(ctx, req.Identity)
	} else {
		user, err = s.repo.GetByPhone(ctx, req.Identity)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	if !models.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, errors.New("invalid credentials")
	}

	version := user.UpdatedAt.UnixNano()
	if user.UpdatedAt.IsZero() {
		version = 0
	}

	accessToken, _, err := s.tokenMaker.CreateToken(user.ID, 24*time.Hour, version, security.TokenScopeAccess)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken: accessToken,
		User: Response{
			ID:        user.ID,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
			Phone:     user.Phone,
			CreatedAt: user.CreatedAt,
		},
	}, nil
}

func (s *service) RequestPasswordReset(ctx context.Context, email string) error {
	// In a real application, you would generate a unique, short-lived token,
	// store it with the user's ID, and email a link containing the token.
	// For this example, we'll just log that a request was made.
	_, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if the user exists or not
		return nil
	}
	// TODO: Implement token generation and email sending
	return nil
}

func (s *service) ResetPassword(_ context.Context, _, _ string) error {
	// TODO: Implement password reset logic
	// 1. Validate the token
	// 2. Find the user associated with the token
	// 3. Hash the new password
	// 4. Update the user's password in the database
	// 5. Invalidate the reset token
	return nil
}

func (s *service) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return s.repo.AssignRole(ctx, userID, roleID)
}
