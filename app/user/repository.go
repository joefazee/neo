package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

func (r *repository) GetByIDWithPermissions(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Preload("Roles.Permissions").
		First(&user, "id = ?", userID).Error
	return &user, err
}

// GetByID returns a user by their ID.
func (r *repository) GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, "id = ?", userID).Error
	return &user, err
}

// GetUsers retrieves a paginated and filtered list of users.
func (r *repository) GetUsers(ctx context.Context, filters *AdminUserFilters) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := r.db.WithContext(ctx).Model(&models.User{})

	if filters.Status != "" {
		isActive := filters.Status == "active"
		query = query.Where("is_active = ?", isActive)
	}

	if filters.Search != "" {
		searchTerm := "%" + strings.ToLower(filters.Search) + "%"
		query = query.Where("first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ?", searchTerm, searchTerm, searchTerm)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting users: %w", err)
	}

	// Apply pagination
	offset := (filters.Page - 1) * filters.PerPage
	query = query.Offset(offset).Limit(filters.PerPage)

	// Apply sorting
	if filters.SortBy != "" && filters.SortOrder != "" {
		query = query.Order(fmt.Sprintf("%s %s", filters.SortBy, filters.SortOrder))
	} else {
		query = query.Order("created_at DESC")
	}

	err := query.Find(&users).Error
	return users, total, err
}

// UpdateUserStatus updates the active status of a single user.
func (r *repository) UpdateUserStatus(ctx context.Context, userID uuid.UUID, isActive bool) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("is_active", isActive).Error
}

// AssignRole assigns a role to a user. It removes existing roles and assigns the new one.
func (r *repository) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM user_roles WHERE user_id = ?", userID).Error; err != nil {
			return err
		}
		return tx.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", userID, roleID).Error
	})
}

// BulkAssignPermissions assigns a list of permissions to multiple users.
func (r *repository) BulkAssignPermissions(ctx context.Context, userIDs, permissionIDs []uuid.UUID) error {
	if len(userIDs) == 0 || len(permissionIDs) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var users []models.User
		if err := tx.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return fmt.Errorf("failed to find users for bulk permission assignment: %w", err)
		}

		var permissions []models.Permission
		if err := tx.Where("id IN ?", permissionIDs).Find(&permissions).Error; err != nil {
			return fmt.Errorf("failed to find permissions for bulk assignment: %w", err)
		}

		if len(permissions) == 0 {
			return errors.New("no valid permissions found to assign")
		}

		for i := range users {
			user := users[i]
			if err := tx.Model(&user).Association("Permissions").Append(&permissions); err != nil {
				return fmt.Errorf("failed to assign permissions to user %s: %w", user.ID, err)
			}
		}

		return nil
	})
}
