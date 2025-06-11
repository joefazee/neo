package user

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joefazee/neo/internal/validator"
	"github.com/joefazee/neo/models"
	"github.com/stretchr/testify/assert"
)

type fakeStripper struct{}

func (fakeStripper) StripHTML(s string) string {
	return strings.ToUpper(s)
}

type identityStripper struct{}

func (identityStripper) StripHTML(s string) string {
	return s
}

func TestAdminUserFilters_SanitizeAndValidate(t *testing.T) {
	f := &AdminUserFilters{
		Search:    "search",
		Status:    "",
		SortBy:    "",
		SortOrder: "",
	}
	v := validator.New()
	f.SanitizeAndValidate(v, fakeStripper{})
	assert.True(t, v.Valid())
	assert.Equal(t, "SEARCH", f.Search)
	assert.Equal(t, "", f.Status)
	assert.Equal(t, "", f.SortBy)
	assert.Equal(t, "", f.SortOrder)

	f2 := &AdminUserFilters{
		Search:    "keep",
		Status:    "invalid",
		SortBy:    "wrong",
		SortOrder: "bad",
	}
	v2 := validator.New()
	f2.SanitizeAndValidate(v2, identityStripper{})
	assert.False(t, v2.Valid())
	assert.Equal(t, "status must be either active or inactive", v2.Errors["status"])
	assert.Equal(t, "invalid sort field", v2.Errors["sort_by"])
	assert.Equal(t, "sort order must be either asc or desc", v2.Errors["sort_order"])
}

func TestAdminAssignRoleRequest_Validate(t *testing.T) {
	r := &AdminAssignRoleRequest{RoleID: uuid.Nil}
	v := validator.New()
	r.Validate(v)
	assert.False(t, v.Valid())
	assert.Equal(t, "role_id is required", v.Errors["role_id"])

	r2 := &AdminAssignRoleRequest{RoleID: uuid.New()}
	v2 := validator.New()
	r2.Validate(v2)
	assert.True(t, v2.Valid())
}

func TestAdminUpdateUserStatusRequest_Validate(t *testing.T) {
	r := &AdminUpdateUserStatusRequest{IsActive: nil}
	v := validator.New()
	r.Validate(v)
	assert.False(t, v.Valid())
	assert.Equal(t, "is_active is a required field", v.Errors["is_active"])

	active := true
	r2 := &AdminUpdateUserStatusRequest{IsActive: &active}
	v2 := validator.New()
	r2.Validate(v2)
	assert.True(t, v2.Valid())
}

func TestAdminBulkAssignRequest_Validate(t *testing.T) {
	r := &AdminBulkAssignRequest{
		UserIDs:       []uuid.UUID{},
		PermissionIDs: []uuid.UUID{},
	}
	v := validator.New()
	r.Validate(v)
	assert.False(t, v.Valid())
	assert.Equal(t, "At least one user_id is required", v.Errors["user_ids"])
	assert.Equal(t, "At least one permission_id is required", v.Errors["permission_ids"])

	r2 := &AdminBulkAssignRequest{
		UserIDs:       []uuid.UUID{uuid.New()},
		PermissionIDs: []uuid.UUID{uuid.New()},
	}
	v2 := validator.New()
	r2.Validate(v2)
	assert.True(t, v2.Valid())
}

func TestToPermissionResponse(t *testing.T) {
	t1 := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
	perm := &models.Permission{
		ID:          uuid.New(),
		Name:        "perm",
		Description: "desc",
		CreatedAt:   t1,
		UpdatedAt:   t1,
	}
	resp := ToPermissionResponse(perm)
	assert.Equal(t, perm.ID, resp.ID)
	assert.Equal(t, perm.Name, resp.Name)
	assert.Equal(t, perm.Description, resp.Description)
	assert.Equal(t, perm.CreatedAt, resp.CreatedAt)
	assert.Equal(t, perm.UpdatedAt, resp.UpdatedAt)
}

func TestToRoleResponse(t *testing.T) {
	t1 := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
	perms := []models.Permission{
		{ID: uuid.New(), Name: "p1", Description: "d1", CreatedAt: t1, UpdatedAt: t1},
		{ID: uuid.New(), Name: "p2", Description: "d2", CreatedAt: t1, UpdatedAt: t1},
	}
	role := &models.Role{
		ID:          uuid.New(),
		Name:        "role",
		Description: "rdesc",
		Permissions: perms,
		CreatedAt:   t1,
		UpdatedAt:   t1,
	}
	resp := ToRoleResponse(role)
	assert.Equal(t, role.ID, resp.ID)
	assert.Equal(t, role.Name, resp.Name)
	assert.Equal(t, role.Description, resp.Description)
	assert.Equal(t, role.CreatedAt, resp.CreatedAt)
	assert.Equal(t, role.UpdatedAt, resp.UpdatedAt)
	assert.Len(t, resp.Permissions, len(perms))
	for i, p := range perms {
		assert.Equal(t, p.ID, resp.Permissions[i].ID)
		assert.Equal(t, p.Name, resp.Permissions[i].Name)
		assert.Equal(t, p.Description, resp.Permissions[i].Description)
		assert.Equal(t, p.CreatedAt, resp.Permissions[i].CreatedAt)
		assert.Equal(t, p.UpdatedAt, resp.Permissions[i].UpdatedAt)
	}
}

func TestToUserResponse(t *testing.T) {
	t1 := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
	roleModel := models.Role{
		ID:          uuid.New(),
		Name:        "role",
		Description: "rdesc",
		Permissions: []models.Permission{
			{ID: uuid.New(), Name: "p", Description: "d", CreatedAt: t1, UpdatedAt: t1},
		},
		CreatedAt: t1,
		UpdatedAt: t1,
	}
	user := &models.User{
		ID:              uuid.New(),
		Email:           "e",
		EmailVerifiedAt: &t1,
		FirstName:       "fn",
		LastName:        "ln",
		Phone:           "ph",
		PhoneVerifiedAt: &t1,
		DateOfBirth:     &t1,
		KYCStatus:       models.KYCStatus("verified"),
		KYCVerifiedAt:   &t1,
		LastLoginAt:     &t1,
		IsActive:        ptrBool(true),
		Roles:           []models.Role{roleModel},
		Country:         nil,
		CreatedAt:       t1,
		UpdatedAt:       t1,
	}
	resp := ToUserResponse(user)
	assert.Equal(t, user.ID, resp.ID)
	assert.Equal(t, user.Email, resp.Email)
	assert.Equal(t, *user.EmailVerifiedAt, *resp.EmailVerifiedAt)
	assert.Equal(t, user.FirstName, resp.FirstName)
	assert.Equal(t, user.LastName, resp.LastName)
	assert.Equal(t, user.Phone, resp.Phone)
	assert.Equal(t, *user.PhoneVerifiedAt, *resp.PhoneVerifiedAt)
	assert.Equal(t, *user.DateOfBirth, *resp.DateOfBirth)
	assert.Equal(t, user.KYCStatus, resp.KYCStatus)
	assert.Equal(t, *user.KYCVerifiedAt, *resp.KYCVerifiedAt)
	assert.Equal(t, *user.LastLoginAt, *resp.LastLoginAt)
	assert.Equal(t, *user.IsActive, *resp.IsActive)
	assert.Len(t, resp.Roles, 1)
	assert.Nil(t, resp.Country)
	assert.Equal(t, user.CreatedAt, resp.CreatedAt)
	assert.Equal(t, user.UpdatedAt, resp.UpdatedAt)

	country := &models.Country{
		ID:             uuid.New(),
		Name:           "name",
		Code:           "code",
		CurrencyCode:   "cc",
		CurrencySymbol: "cs",
	}
	user.Country = country
	resp2 := ToUserResponse(user)
	assert.NotNil(t, resp2.Country)
	assert.Equal(t, country.ID, resp2.Country.ID)
	assert.Equal(t, country.Name, resp2.Country.Name)
	assert.Equal(t, country.Code, resp2.Country.Code)
	assert.Equal(t, country.CurrencyCode, resp2.Country.CurrencyCode)
	assert.Equal(t, country.CurrencySymbol, resp2.Country.CurrencySymbol)
}

func ptrBool(b bool) *bool {
	return &b
}
