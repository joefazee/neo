package user

import (
	"context"

	"github.com/google/uuid"

	"github.com/joefazee/neo/models"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new user repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *repository) GetByPhone(ctx context.Context, phone string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error
	return &user, err
}

func (r *repository) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *repository) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", userID, roleID).Error
}

func (r *repository) GetByIDWithPermissions(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Preload("Roles.Permissions").
		First(&user, "id = ?", userID).Error
	return &user, err
}
