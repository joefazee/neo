package user

import (
	"time"

	"github.com/joefazee/neo/models"

	"github.com/google/uuid"
	"github.com/joefazee/neo/internal/sanitizer"
	"github.com/joefazee/neo/internal/validator"
)

// AdminUserFilters defines the query parameters for filtering the user list.
type AdminUserFilters struct {
	Page      int    `form:"page"`
	PerPage   int    `form:"per_page"`
	Status    string `form:"status"` // "active" or "inactive"
	Search    string `form:"search"`
	SortBy    string `form:"sort_by"`
	SortOrder string `form:"sort_order"`
}

// SanitizeAndValidate cleans and validates the filter inputs.
func (f *AdminUserFilters) SanitizeAndValidate(v *validator.Validator, s sanitizer.HTMLStripperer) {
	f.Search = s.StripHTML(f.Search)
	f.Status = s.StripHTML(f.Status)
	f.SortBy = s.StripHTML(f.SortBy)
	f.SortOrder = s.StripHTML(f.SortOrder)

	v.Check(validator.In(f.Status, "", "active", "inactive"), "status", "status must be either active or inactive")
	v.Check(validator.In(f.SortBy, "", "created_at", "first_name", "email"), "sort_by", "invalid sort field")
	v.Check(validator.In(f.SortOrder, "", "asc", "desc"), "sort_order", "sort order must be either asc or desc")
}

// AdminAssignRoleRequest is the request body for assigning a role to a user.
type AdminAssignRoleRequest struct {
	RoleID uuid.UUID `json:"role_id"`
}

// Validate checks the request data.
func (r *AdminAssignRoleRequest) Validate(v *validator.Validator) {
	v.Check(r.RoleID != uuid.Nil, "role_id", "role_id is required")
}

// AdminUpdateUserStatusRequest is the request body for changing a user's active status.
type AdminUpdateUserStatusRequest struct {
	IsActive *bool `json:"is_active"`
}

// Validate checks the request data.
func (r *AdminUpdateUserStatusRequest) Validate(v *validator.Validator) {
	v.Check(r.IsActive != nil, "is_active", "is_active is a required field")
}

// AdminBulkAssignRequest is the request body for bulk assigning roles or permissions.
type AdminBulkAssignRequest struct {
	UserIDs       []uuid.UUID `json:"user_ids"`
	PermissionIDs []uuid.UUID `json:"permission_ids"`
}

// Validate checks the request data.
func (r *AdminBulkAssignRequest) Validate(v *validator.Validator) {
	v.Check(len(r.UserIDs) > 0, "user_ids", "At least one user_id is required")
	v.Check(len(r.PermissionIDs) > 0, "permission_ids", "At least one permission_id is required")
}

// AdminUserResponse is the detailed user response for admin views.
type AdminUserResponse struct {
	ID              uuid.UUID        `json:"id"`
	Email           string           `json:"email"`
	EmailVerifiedAt *time.Time       `json:"email_verified_at"`
	FirstName       string           `json:"first_name"`
	LastName        string           `json:"last_name"`
	Phone           string           `json:"phone"`
	PhoneVerifiedAt *time.Time       `json:"phone_verified_at"`
	DateOfBirth     *time.Time       `json:"date_of_birth"`
	KYCStatus       models.KYCStatus `json:"kyc_status"`
	KYCVerifiedAt   *time.Time       `json:"kyc_verified_at"`
	LastLoginAt     *time.Time       `json:"last_login_at"`
	IsActive        *bool            `json:"is_active"`
	Roles           []*RoleResponse  `json:"roles"`
	Country         *CountryInfo     `json:"country,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type CountryInfo struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Code           string    `json:"code"`
	CurrencyCode   string    `json:"currency_code"`
	CurrencySymbol string    `json:"currency_symbol"`
}

// CreatePermissionRequest represents the request to create a permission
type CreatePermissionRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=50"`
	Description string `json:"description,omitempty"`
}

// CreateRoleRequest represents the request to create a role
type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=50"`
	Description string `json:"description,omitempty"`
}

// UpdateRoleRequest represents the request to update a role
type UpdateRoleRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=2,max=50"`
	Description *string `json:"description,omitempty"`
}

// AssignPermissionsRequest represents the request to assign permissions to a role
type AssignPermissionsRequest struct {
	PermissionCodes []string `json:"permission_codes" binding:"required,min=1"`
}

// RemovePermissionsRequest represents the request to remove permissions from a role
type RemovePermissionsRequest struct {
	PermissionCodes []string `json:"permission_codes" binding:"required,min=1"`
}

// PermissionResponse represents a permission in API responses
type PermissionResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RoleResponse represents a role in API responses
type RoleResponse struct {
	ID          uuid.UUID            `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Permissions []PermissionResponse `json:"permissions"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

// ToPermissionResponse converts a models.Permission to PermissionResponse
func ToPermissionResponse(permission *models.Permission) *PermissionResponse {
	return &PermissionResponse{
		ID:          permission.ID,
		Name:        permission.Name,
		Description: permission.Description,
		CreatedAt:   permission.CreatedAt,
		UpdatedAt:   permission.UpdatedAt,
	}
}

// ToRoleResponse converts a models.Role to RoleResponse
func ToRoleResponse(role *models.Role) *RoleResponse {
	permissions := make([]PermissionResponse, len(role.Permissions))
	for i, perm := range role.Permissions {
		permissions[i] = *ToPermissionResponse(&perm)
	}

	return &RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		Permissions: permissions,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
}

// ToUserResponse converts a models.User to UserResponse
func ToUserResponse(user *models.User) *AdminUserResponse {
	roles := make([]*RoleResponse, len(user.Roles))
	for i, role := range user.Roles {
		roles[i] = ToRoleResponse(&role)
	}

	response := &AdminUserResponse{
		ID:              user.ID,
		Email:           user.Email,
		EmailVerifiedAt: user.EmailVerifiedAt,
		FirstName:       user.FirstName,
		LastName:        user.LastName,
		Phone:           user.Phone,
		PhoneVerifiedAt: user.PhoneVerifiedAt,
		DateOfBirth:     user.DateOfBirth,
		KYCStatus:       user.KYCStatus,
		KYCVerifiedAt:   user.KYCVerifiedAt,
		LastLoginAt:     user.LastLoginAt,
		IsActive:        user.IsActive,
		Roles:           roles,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}

	if user.Country != nil {
		response.Country = &CountryInfo{
			ID:             user.Country.ID,
			Name:           user.Country.Name,
			Code:           user.Country.Code,
			CurrencyCode:   user.Country.CurrencyCode,
			CurrencySymbol: user.Country.CurrencySymbol,
		}
	}

	return response
}
