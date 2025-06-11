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
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *repository) GetByPhone(ctx context.Context, phone string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, err
	}
	return &user, nil
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
		// Verify users exist
		var userCount int64
		if err := tx.Model(&models.User{}).Where("id IN ?", userIDs).Count(&userCount).Error; err != nil {
			return fmt.Errorf("failed to count users: %w", err)
		}
		if userCount != int64(len(userIDs)) {
			return errors.New("some users not found")
		}

		// Verify permissions exist
		var permissions []models.Permission
		if err := tx.Where("id IN ?", permissionIDs).Find(&permissions).Error; err != nil {
			return fmt.Errorf("failed to find permissions: %w", err)
		}
		if len(permissions) == 0 {
			return errors.New("no valid permissions found to assign")
		}

		// For each user, create a personal role and assign permissions
		for _, userID := range userIDs {
			roleName := fmt.Sprintf("user_%s_bulk_permissions", userID.String())
			role := &models.Role{
				Name:        roleName,
				Description: "Bulk assigned permissions",
			}

			if err := tx.Create(role).Error; err != nil {
				return fmt.Errorf("failed to create role for user %s: %w", userID, err)
			}

			if err := tx.Model(role).Association("Permissions").Append(permissions); err != nil {
				return fmt.Errorf("failed to assign permissions to role for user %s: %w", userID, err)
			}

			if err := tx.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", userID, role.ID).Error; err != nil {
				return fmt.Errorf("failed to assign role to user %s: %w", userID, err)
			}
		}

		return nil
	})
}
func (r *repository) CreatePermission(ctx context.Context, permission *models.Permission) error {
	return r.db.WithContext(ctx).Create(permission).Error
}

func (r *repository) GetPermissionByName(ctx context.Context, name string) (*models.Permission, error) {
	var permission models.Permission
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&permission).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

func (r *repository) GetPermissionsByNames(ctx context.Context, names []string) ([]models.Permission, error) {
	var permissions []models.Permission
	err := r.db.WithContext(ctx).Where("name IN ?", names).Find(&permissions).Error
	return permissions, err
}

func (r *repository) CreateRole(ctx context.Context, role *models.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *repository) UpdateRole(ctx context.Context, role *models.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *repository) GetRoleByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).Preload("Permissions").Where("id = ?", id).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// AssignPermissionsToRole assigns a list of permissions to a role.
// //nolint: dupl
func (r *repository) AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Verify role exists
		var roleExists bool
		err := tx.Model(&models.Role{}).Select("count(*) > 0").Where("id = ?", roleID).Find(&roleExists).Error
		if err != nil {
			return err
		}
		if !roleExists {
			return fmt.Errorf("role %s does not exist", roleID)
		}

		// Verify all permissions exist
		var permCount int64
		err = tx.Model(&models.Permission{}).Where("id IN ?", permissionIDs).Count(&permCount).Error
		if err != nil {
			return err
		}
		if permCount != int64(len(permissionIDs)) {
			return fmt.Errorf("some permissions do not exist")
		}

		for _, permID := range permissionIDs {
			err = tx.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?) ON CONFLICT DO NOTHING", roleID, permID).Error
			if err != nil {
				return fmt.Errorf("failed to insert role_permission: %w", err)
			}
		}

		return nil
	})
}

// RemovePermissionsFromRole removes a list of permissions from a role.
// //nolint: dupl
func (r *repository) RemovePermissionsFromRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	var role models.Role
	if err := r.db.WithContext(ctx).First(&role, roleID).Error; err != nil {
		return err
	}

	var permissions []models.Permission
	if err := r.db.WithContext(ctx).Where("id IN ?", permissionIDs).Find(&permissions).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).Model(&role).Association("Permissions").Delete(permissions)
}

func (r *repository) GetUserByIDWithRoles(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Preload("Roles").
		Preload("Roles.Permissions").
		Preload("Country").
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return err
	}

	var role models.Role
	if err := r.db.WithContext(ctx).First(&role, roleID).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).Model(&user).Association("Roles").Delete(&role)
}
