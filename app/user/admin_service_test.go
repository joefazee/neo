package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/joefazee/neo/models"
)

type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	if args.Get(0) != nil {
		return args.Error(0)
	}
	return nil
}

func (m *MockRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if u := args.Get(0); u != nil {
		return u.(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepo) GetByPhone(ctx context.Context, phone string) (*models.User, error) {
	args := m.Called(ctx, phone)
	if u := args.Get(0); u != nil {
		return u.(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepo) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	if args.Get(0) != nil {
		return args.Error(0)
	}
	return nil
}

func (m *MockRepo) GetByIDWithPermissions(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, userID)
	if u := args.Get(0); u != nil {
		return u.(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepo) GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, userID)
	if u := args.Get(0); u != nil {
		return u.(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepo) GetUsers(ctx context.Context, filters *AdminUserFilters) ([]models.User, int64, error) {
	args := m.Called(ctx, filters)
	if users := args.Get(0); users != nil {
		return users.([]models.User), args.Get(1).(int64), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *MockRepo) UpdateUserStatus(ctx context.Context, userID uuid.UUID, isActive bool) error {
	return m.Called(ctx, userID, isActive).Error(0)
}

func (m *MockRepo) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return m.Called(ctx, userID, roleID).Error(0)
}

func (m *MockRepo) BulkAssignPermissions(ctx context.Context, userIDs, permissionIDs []uuid.UUID) error {
	return m.Called(ctx, userIDs, permissionIDs).Error(0)
}

func (m *MockRepo) GetPermissionByName(ctx context.Context, name string) (*models.Permission, error) {
	args := m.Called(ctx, name)
	if p := args.Get(0); p != nil {
		return p.(*models.Permission), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepo) CreatePermission(ctx context.Context, permission *models.Permission) error {
	return m.Called(ctx, permission).Error(0)
}

func (m *MockRepo) CreateRole(ctx context.Context, role *models.Role) error {
	return m.Called(ctx, role).Error(0)
}

func (m *MockRepo) GetRoleByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	args := m.Called(ctx, id)
	if r := args.Get(0); r != nil {
		return r.(*models.Role), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepo) UpdateRole(ctx context.Context, role *models.Role) error {
	return m.Called(ctx, role).Error(0)
}

func (m *MockRepo) GetPermissionsByNames(ctx context.Context, codes []string) ([]models.Permission, error) {
	args := m.Called(ctx, codes)
	if perms := args.Get(0); perms != nil {
		return perms.([]models.Permission), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepo) AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	return m.Called(ctx, roleID, permissionIDs).Error(0)
}

func (m *MockRepo) RemovePermissionsFromRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	return m.Called(ctx, roleID, permissionIDs).Error(0)
}

func (m *MockRepo) GetUserByIDWithRoles(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if u := args.Get(0); u != nil {
		return u.(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepo) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return m.Called(ctx, userID, roleID).Error(0)
}

// Helper functions
func ptrString(s string) *string { return &s }
func TestGetUsers(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)

	// Use filters that will be normalized by the service
	filters := &AdminUserFilters{Page: 0, PerPage: 200}
	// Expected normalized filters
	expectedFilters := &AdminUserFilters{Page: 1, PerPage: 20}

	users := []models.User{{
		ID:        uuid.New(),
		FirstName: "A",
		LastName:  "B",
		Email:     "e",
		Phone:     "p",
		IsActive:  ptrBool(true),
		CreatedAt: time.Now(),
		Roles:     []models.Role{},
	}}

	// Mock expects the normalized filters
	repo.On("GetUsers", mock.Anything, expectedFilters).Return(users, int64(1), nil)

	res, total, err := svc.GetUsers(context.Background(), filters)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, res, 1)
	assert.Equal(t, users[0].ID, res[0].ID)

	repo.AssertExpectations(t)
}

func TestGetUsers_Error(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	filters := &AdminUserFilters{Page: 1, PerPage: 20}

	repo.On("GetUsers", mock.Anything, filters).Return(nil, int64(0), errors.New("fail"))

	_, _, err := svc.GetUsers(context.Background(), filters)
	assert.EqualError(t, err, "fail")

	repo.AssertExpectations(t)
}

func TestUpdateUserStatus(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	id := uuid.New()

	repo.On("UpdateUserStatus", mock.Anything, id, true).Return(nil)

	err := svc.UpdateUserStatus(context.Background(), id, true)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestUpdateUserStatus_Error(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	id := uuid.New()

	repo.On("UpdateUserStatus", mock.Anything, id, false).Return(errors.New("err"))

	err := svc.UpdateUserStatus(context.Background(), id, false)
	assert.EqualError(t, err, "err")

	repo.AssertExpectations(t)
}

func TestAssignRole(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	u, r := uuid.New(), uuid.New()

	repo.On("AssignRole", mock.Anything, u, r).Return(nil)

	err := svc.AssignRole(context.Background(), u, r)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestAssignRole_Error(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	u, r := uuid.New(), uuid.New()

	repo.On("AssignRole", mock.Anything, u, r).Return(errors.New("fail"))

	err := svc.AssignRole(context.Background(), u, r)
	assert.EqualError(t, err, "fail")

	repo.AssertExpectations(t)
}

func TestBulkAssignPermissions(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	uids, pids := []uuid.UUID{uuid.New()}, []uuid.UUID{uuid.New()}

	repo.On("BulkAssignPermissions", mock.Anything, uids, pids).Return(nil)

	err := svc.BulkAssignPermissions(context.Background(), uids, pids)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestBulkAssignPermissions_Error(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	uids, pids := []uuid.UUID{uuid.New()}, []uuid.UUID{uuid.New()}

	repo.On("BulkAssignPermissions", mock.Anything, uids, pids).Return(errors.New("err"))

	err := svc.BulkAssignPermissions(context.Background(), uids, pids)
	assert.EqualError(t, err, "err")

	repo.AssertExpectations(t)
}

func TestCreatePermission(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	req := &CreatePermissionRequest{Name: "test_permission", Description: "Test Description"}

	repo.On("GetPermissionByName", mock.Anything, req.Name).Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreatePermission", mock.Anything, mock.MatchedBy(func(p *models.Permission) bool {
		return p.Name == req.Name && p.Description == req.Description
	})).Run(func(args mock.Arguments) {
		// Simulate setting ID on creation
		p := args.Get(1).(*models.Permission)
		p.ID = uuid.New()
		p.CreatedAt = time.Now()
		p.UpdatedAt = time.Now()
	}).Return(nil)

	resp, err := svc.CreatePermission(context.Background(), req)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, req.Name, resp.Name)
	assert.Equal(t, req.Description, resp.Description)

	repo.AssertExpectations(t)
}

func TestCreatePermission_Exists(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	existing := &models.Permission{ID: uuid.New(), Name: "existing"}

	repo.On("GetPermissionByName", mock.Anything, "existing").Return(existing, nil)

	resp, err := svc.CreatePermission(context.Background(), &CreatePermissionRequest{Name: "existing"})
	assert.Nil(t, resp)
	assert.EqualError(t, err, "permission with this name already exists")

	repo.AssertExpectations(t)
}

func TestCreatePermission_ErrorOnCheck(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)

	repo.On("GetPermissionByName", mock.Anything, "test").Return(nil, errors.New("db error"))

	_, err := svc.CreatePermission(context.Background(), &CreatePermissionRequest{Name: "test"})
	assert.EqualError(t, err, "failed to check existing permission: db error")

	repo.AssertExpectations(t)
}

func TestCreatePermission_ErrorOnCreate(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)

	repo.On("GetPermissionByName", mock.Anything, "test").Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreatePermission", mock.Anything, mock.Anything).Return(errors.New("create error"))

	_, err := svc.CreatePermission(context.Background(), &CreatePermissionRequest{Name: "test"})
	assert.EqualError(t, err, "failed to create permission: create error")

	repo.AssertExpectations(t)
}

func TestCreateRole(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	req := &CreateRoleRequest{Name: "test_role", Description: "Test Description"}

	repo.On("CreateRole", mock.Anything, mock.MatchedBy(func(r *models.Role) bool {
		return r.Name == req.Name && r.Description == req.Description
	})).Run(func(args mock.Arguments) {
		// Simulate setting ID on creation
		r := args.Get(1).(*models.Role)
		r.ID = uuid.New()
		r.CreatedAt = time.Now()
		r.UpdatedAt = time.Now()
	}).Return(nil)

	resp, err := svc.CreateRole(context.Background(), req)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, req.Name, resp.Name)
	assert.Equal(t, req.Description, resp.Description)

	repo.AssertExpectations(t)
}

func TestCreateRole_Error(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)

	repo.On("CreateRole", mock.Anything, mock.Anything).Return(errors.New("create error"))

	_, err := svc.CreateRole(context.Background(), &CreateRoleRequest{Name: "test"})
	assert.EqualError(t, err, "failed to create role: create error")

	repo.AssertExpectations(t)
}

func TestUpdateRole_NotFound(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	id := uuid.New()

	repo.On("GetRoleByID", mock.Anything, id).Return(nil, gorm.ErrRecordNotFound)

	_, err := svc.UpdateRole(context.Background(), id, &UpdateRoleRequest{})
	assert.Equal(t, models.ErrRecordNotFound, err)

	repo.AssertExpectations(t)
}

func TestUpdateRole_ErrorOnGet(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	id := uuid.New()

	repo.On("GetRoleByID", mock.Anything, id).Return(nil, errors.New("get error"))

	_, err := svc.UpdateRole(context.Background(), id, &UpdateRoleRequest{})
	assert.EqualError(t, err, "failed to get role: get error")

	repo.AssertExpectations(t)
}

func TestUpdateRole_ErrorOnUpdate(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	id := uuid.New()
	role := &models.Role{ID: id, Name: "old", Description: "old"}

	repo.On("GetRoleByID", mock.Anything, id).Return(role, nil)
	repo.On("UpdateRole", mock.Anything, mock.MatchedBy(func(r *models.Role) bool {
		return r.ID == id && r.Name == "new"
	})).Return(errors.New("update error"))

	_, err := svc.UpdateRole(context.Background(), id, &UpdateRoleRequest{Name: ptrString("new")})
	assert.EqualError(t, err, "failed to update role: update error")

	repo.AssertExpectations(t)
}

func TestUpdateRole_Success(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	id := uuid.New()
	role := &models.Role{
		ID:          id,
		Name:        "old",
		Description: "old",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	repo.On("GetRoleByID", mock.Anything, id).Return(role, nil)
	repo.On("UpdateRole", mock.Anything, mock.MatchedBy(func(r *models.Role) bool {
		return r.ID == id && r.Name == "new" && r.Description == "new_desc"
	})).Return(nil)

	resp, err := svc.UpdateRole(context.Background(), id, &UpdateRoleRequest{
		Name:        ptrString("new"),
		Description: ptrString("new_desc"),
	})
	assert.NoError(t, err)
	assert.Equal(t, id, resp.ID)
	assert.Equal(t, "new", resp.Name)
	assert.Equal(t, "new_desc", resp.Description)

	repo.AssertExpectations(t)
}

func TestAssignPermissionsToRole(t *testing.T) {
	t.Run("error getting permissions", func(t *testing.T) {
		repo := &MockRepo{}
		svc := NewAdminService(repo)
		rid := uuid.New()
		codes := []string{"c1", "c2"}

		repo.On("GetPermissionsByNames", mock.Anything, codes).Return(nil, errors.New("get error"))

		_, err := svc.AssignPermissionsToRole(context.Background(), rid, &AssignPermissionsRequest{PermissionCodes: codes})
		assert.EqualError(t, err, "failed to get permissions: get error")

		repo.AssertExpectations(t)
	})

	t.Run("permission codes not found", func(t *testing.T) {
		repo := &MockRepo{}
		svc := NewAdminService(repo)
		rid := uuid.New()

		repo.On("GetPermissionsByNames", mock.Anything, []string{"c1", "c2", "c3"}).Return([]models.Permission{{ID: uuid.New()}, {ID: uuid.New()}}, nil)

		_, err := svc.AssignPermissionsToRole(context.Background(), rid, &AssignPermissionsRequest{PermissionCodes: []string{"c1", "c2", "c3"}})
		assert.EqualError(t, err, "one or more permission codes not found")

		repo.AssertExpectations(t)
	})

	t.Run("error assigning permissions", func(t *testing.T) {
		repo := &MockRepo{}
		svc := NewAdminService(repo)
		rid := uuid.New()
		codes := []string{"c1", "c2"}
		perms := []models.Permission{{ID: uuid.New()}, {ID: uuid.New()}}
		ids := []uuid.UUID{perms[0].ID, perms[1].ID}

		repo.On("GetPermissionsByNames", mock.Anything, codes).Return(perms, nil)
		repo.On("AssignPermissionsToRole", mock.Anything, rid, ids).Return(errors.New("assign error"))

		_, err := svc.AssignPermissionsToRole(context.Background(), rid, &AssignPermissionsRequest{PermissionCodes: codes})
		assert.EqualError(t, err, "failed to assign permissions to role: assign error")

		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		repo := &MockRepo{}
		svc := NewAdminService(repo)
		rid := uuid.New()
		codes := []string{"c1", "c2"}
		perms := []models.Permission{
			{ID: uuid.New(), Name: "c1", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: uuid.New(), Name: "c2", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}
		ids := []uuid.UUID{perms[0].ID, perms[1].ID}
		updatedRole := &models.Role{
			ID:          rid,
			Name:        "test_role",
			Description: "Test Role",
			Permissions: perms,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		repo.On("GetPermissionsByNames", mock.Anything, codes).Return(perms, nil)
		repo.On("AssignPermissionsToRole", mock.Anything, rid, ids).Return(nil)
		repo.On("GetRoleByID", mock.Anything, rid).Return(updatedRole, nil)

		resp, err := svc.AssignPermissionsToRole(context.Background(), rid, &AssignPermissionsRequest{PermissionCodes: codes})
		assert.NoError(t, err)
		assert.Equal(t, rid, resp.ID)
		assert.Len(t, resp.Permissions, len(perms))

		repo.AssertExpectations(t)
	})
}

func TestRemovePermissionsFromRole(t *testing.T) {
	t.Run("error getting permissions", func(t *testing.T) {
		repo := &MockRepo{}
		svc := NewAdminService(repo)
		rid := uuid.New()
		codes := []string{"c1"}

		repo.On("GetPermissionsByNames", mock.Anything, codes).Return(nil, errors.New("get error"))

		_, err := svc.RemovePermissionsFromRole(context.Background(), rid, &RemovePermissionsRequest{PermissionCodes: codes})
		assert.EqualError(t, err, "failed to get permissions: get error")

		repo.AssertExpectations(t)
	})

	t.Run("permission codes not found", func(t *testing.T) {
		repo := &MockRepo{}
		svc := NewAdminService(repo)
		rid := uuid.New()
		codes := []string{"c1"}

		repo.On("GetPermissionsByNames", mock.Anything, codes).Return([]models.Permission{}, nil)

		_, err := svc.RemovePermissionsFromRole(context.Background(), rid, &RemovePermissionsRequest{PermissionCodes: codes})
		assert.EqualError(t, err, "one or more permission codes not found")

		repo.AssertExpectations(t)
	})

	t.Run("error removing permissions", func(t *testing.T) {
		repo := &MockRepo{}
		svc := NewAdminService(repo)
		rid := uuid.New()
		codes := []string{"c1"}
		perms := []models.Permission{{ID: uuid.New()}}
		ids := []uuid.UUID{perms[0].ID}

		repo.On("GetPermissionsByNames", mock.Anything, codes).Return(perms, nil)
		repo.On("RemovePermissionsFromRole", mock.Anything, rid, ids).Return(errors.New("remove error"))

		_, err := svc.RemovePermissionsFromRole(context.Background(), rid, &RemovePermissionsRequest{PermissionCodes: codes})
		assert.EqualError(t, err, "failed to remove permissions from role: remove error")

		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		repo := &MockRepo{}
		svc := NewAdminService(repo)
		rid := uuid.New()
		codes := []string{"c1"}
		perms := []models.Permission{{ID: uuid.New(), Name: "c1"}}
		ids := []uuid.UUID{perms[0].ID}
		updatedRole := &models.Role{
			ID:          rid,
			Name:        "test_role",
			Description: "Test Role",
			Permissions: []models.Permission{},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		repo.On("GetPermissionsByNames", mock.Anything, codes).Return(perms, nil)
		repo.On("RemovePermissionsFromRole", mock.Anything, rid, ids).Return(nil)
		repo.On("GetRoleByID", mock.Anything, rid).Return(updatedRole, nil)

		resp, err := svc.RemovePermissionsFromRole(context.Background(), rid, &RemovePermissionsRequest{PermissionCodes: codes})
		assert.NoError(t, err)
		assert.Equal(t, rid, resp.ID)
		assert.Empty(t, resp.Permissions)

		repo.AssertExpectations(t)
	})
}

func TestGetUserByID(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		repo := &MockRepo{}
		svc := NewAdminService(repo)
		uid := uuid.New()

		repo.On("GetUserByIDWithRoles", mock.Anything, uid).Return(nil, gorm.ErrRecordNotFound)

		_, err := svc.GetUserByID(context.Background(), uid)
		assert.Equal(t, models.ErrRecordNotFound, err)

		repo.AssertExpectations(t)
	})

	t.Run("database error", func(t *testing.T) {
		repo := &MockRepo{}
		svc := NewAdminService(repo)
		uid := uuid.New()

		repo.On("GetUserByIDWithRoles", mock.Anything, uid).Return(nil, errors.New("db error"))

		_, err := svc.GetUserByID(context.Background(), uid)
		assert.EqualError(t, err, "failed to get user: db error")

		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		repo := &MockRepo{}
		svc := NewAdminService(repo)
		uid := uuid.New()
		usr := &models.User{
			ID:        uid,
			Email:     "test@example.com",
			FirstName: "John",
			LastName:  "Doe",
			Phone:     "+1234567890",
			IsActive:  ptrBool(true),
			Roles:     []models.Role{{ID: uuid.New(), Name: "admin"}},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		repo.On("GetUserByIDWithRoles", mock.Anything, uid).Return(usr, nil)

		resp, err := svc.GetUserByID(context.Background(), uid)
		assert.NoError(t, err)
		assert.Equal(t, uid, resp.ID)
		assert.Equal(t, usr.Email, resp.Email)
		assert.Len(t, resp.Roles, 1)

		repo.AssertExpectations(t)
	})
}

func TestRemoveRoleFromUser(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		repo := &MockRepo{}
		svc := NewAdminService(repo)
		uid, rid := uuid.New(), uuid.New()

		repo.On("GetUserByIDWithRoles", mock.Anything, uid).Return(nil, gorm.ErrRecordNotFound)

		_, err := svc.RemoveRoleFromUser(context.Background(), uid, rid)
		assert.Equal(t, models.ErrRecordNotFound, err)

		repo.AssertExpectations(t)
	})

	t.Run("database error getting user", func(t *testing.T) {
		repo := &MockRepo{}
		svc := NewAdminService(repo)
		uid, rid := uuid.New(), uuid.New()

		repo.On("GetUserByIDWithRoles", mock.Anything, uid).Return(nil, errors.New("db error"))

		_, err := svc.RemoveRoleFromUser(context.Background(), uid, rid)
		assert.EqualError(t, err, "failed to get user: db error")

		repo.AssertExpectations(t)
	})

	t.Run("user does not have role", func(t *testing.T) {
		repo := &MockRepo{}
		svc := NewAdminService(repo)
		uid, rid := uuid.New(), uuid.New()
		user := &models.User{ID: uid, Roles: []models.Role{}}

		repo.On("GetUserByIDWithRoles", mock.Anything, uid).Return(user, nil)

		_, err := svc.RemoveRoleFromUser(context.Background(), uid, rid)
		assert.EqualError(t, err, "user does not have this role")

		repo.AssertExpectations(t)
	})

	t.Run("error removing role", func(t *testing.T) {
		repo := &MockRepo{}
		svc := NewAdminService(repo)
		uid, rid := uuid.New(), uuid.New()
		userWithRole := &models.User{ID: uid, Roles: []models.Role{{ID: rid}}}

		repo.On("GetUserByIDWithRoles", mock.Anything, uid).Return(userWithRole, nil)
		repo.On("RemoveRoleFromUser", mock.Anything, uid, rid).Return(errors.New("remove error"))

		_, err := svc.RemoveRoleFromUser(context.Background(), uid, rid)
		assert.EqualError(t, err, "failed to remove role from user: remove error")

		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		repo := &MockRepo{}
		svc := NewAdminService(repo)
		uid, rid := uuid.New(), uuid.New()
		userWithRole := &models.User{ID: uid, Roles: []models.Role{{ID: rid}}}
		updatedUser := &models.User{
			ID:        uid,
			Email:     "test@example.com",
			FirstName: "John",
			LastName:  "Doe",
			Phone:     "+1234567890",
			IsActive:  ptrBool(true),
			Roles:     []models.Role{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		repo.On("GetUserByIDWithRoles", mock.Anything, uid).Return(userWithRole, nil).Once()
		repo.On("RemoveRoleFromUser", mock.Anything, uid, rid).Return(nil)
		repo.On("GetUserByIDWithRoles", mock.Anything, uid).Return(updatedUser, nil).Once()

		resp, err := svc.RemoveRoleFromUser(context.Background(), uid, rid)
		assert.NoError(t, err)
		assert.Equal(t, uid, resp.ID)
		assert.Empty(t, resp.Roles)

		repo.AssertExpectations(t)
	})
}

func TestGetUsers_WithRoles(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	filters := &AdminUserFilters{Page: 1, PerPage: 20}

	role1 := models.Role{
		ID:          uuid.New(),
		Name:        "admin",
		Description: "Administrator",
		Permissions: []models.Permission{{ID: uuid.New(), Name: "read"}},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	role2 := models.Role{
		ID:          uuid.New(),
		Name:        "user",
		Description: "Regular User",
		Permissions: []models.Permission{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	users := []models.User{{
		ID:        uuid.New(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
		Phone:     "+1234567890",
		IsActive:  ptrBool(true),
		CreatedAt: time.Now(),
		Roles:     []models.Role{role1, role2},
	}}

	repo.On("GetUsers", mock.Anything, filters).Return(users, int64(1), nil)

	res, total, err := svc.GetUsers(context.Background(), filters)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, res, 1)
	assert.Len(t, res[0].Roles, 2)
	assert.Equal(t, "admin", res[0].Roles[0].Name)
	assert.Equal(t, "user", res[0].Roles[1].Name)
	assert.Len(t, res[0].Roles[0].Permissions, 1)
	assert.Empty(t, res[0].Roles[1].Permissions)

	repo.AssertExpectations(t)
}

func TestAssignPermissionsToRole_ErrorGettingUpdatedRole(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	rid := uuid.New()
	codes := []string{"c1"}
	perms := []models.Permission{{ID: uuid.New(), Name: "c1"}}
	ids := []uuid.UUID{perms[0].ID}

	repo.On("GetPermissionsByNames", mock.Anything, codes).Return(perms, nil)
	repo.On("AssignPermissionsToRole", mock.Anything, rid, ids).Return(nil)
	repo.On("GetRoleByID", mock.Anything, rid).Return(nil, errors.New("failed to fetch updated role"))

	_, err := svc.AssignPermissionsToRole(context.Background(), rid, &AssignPermissionsRequest{PermissionCodes: codes})
	assert.EqualError(t, err, "failed to get updated role: failed to fetch updated role")

	repo.AssertExpectations(t)
}

func TestRemovePermissionsFromRole_ErrorGettingUpdatedRole(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	rid := uuid.New()
	codes := []string{"c1"}
	perms := []models.Permission{{ID: uuid.New(), Name: "c1"}}
	ids := []uuid.UUID{perms[0].ID}

	repo.On("GetPermissionsByNames", mock.Anything, codes).Return(perms, nil)
	repo.On("RemovePermissionsFromRole", mock.Anything, rid, ids).Return(nil)
	repo.On("GetRoleByID", mock.Anything, rid).Return(nil, errors.New("failed to fetch updated role"))

	_, err := svc.RemovePermissionsFromRole(context.Background(), rid, &RemovePermissionsRequest{PermissionCodes: codes})
	assert.EqualError(t, err, "failed to get updated role: failed to fetch updated role")

	repo.AssertExpectations(t)
}

func TestRemoveRoleFromUser_ErrorGettingUpdatedUser(t *testing.T) {
	repo := &MockRepo{}
	svc := NewAdminService(repo)
	uid, rid := uuid.New(), uuid.New()
	userWithRole := &models.User{ID: uid, Roles: []models.Role{{ID: rid}}}

	repo.On("GetUserByIDWithRoles", mock.Anything, uid).Return(userWithRole, nil).Once()
	repo.On("RemoveRoleFromUser", mock.Anything, uid, rid).Return(nil)
	repo.On("GetUserByIDWithRoles", mock.Anything, uid).Return(nil, errors.New("failed to fetch updated user")).Once()

	_, err := svc.RemoveRoleFromUser(context.Background(), uid, rid)
	assert.EqualError(t, err, "failed to get updated user: failed to fetch updated user")

	repo.AssertExpectations(t)
}
