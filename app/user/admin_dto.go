package user

import (
	"time"

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
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	Roles     []string  `json:"roles,omitempty"`
}
