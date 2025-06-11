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

	"github.com/joefazee/neo/internal/cache"
	"github.com/joefazee/neo/models"
)

func TestGetUserPermissions_CacheHit(t *testing.T) {
	repo := &MockRepo{}
	mockCache := &cache.MockCache{}
	svc := NewAuthService(repo, mockCache)

	userID := uuid.New()
	cacheKey := "user:" + userID.String() + ":permissions"
	cachedJSON := `["read","write","delete"]`

	mockCache.On("Get", mock.Anything, cacheKey).Return(cachedJSON, nil)

	permissions, err := svc.GetUserPermissions(context.Background(), userID)

	assert.NoError(t, err)
	assert.Equal(t, []string{"read", "write", "delete"}, permissions)

	repo.AssertNotCalled(t, "GetByIDWithPermissions")
	mockCache.AssertExpectations(t)
}

func TestGetUserPermissions_CacheHitInvalidJSON(t *testing.T) {
	repo := &MockRepo{}
	mockCache := &cache.MockCache{}
	svc := NewAuthService(repo, mockCache)

	userID := uuid.New()
	cacheKey := "user:" + userID.String() + ":permissions"
	invalidJSON := `invalid json`

	user := &models.User{
		ID: userID,
		Roles: []models.Role{
			{
				ID:   uuid.New(),
				Name: "admin",
				Permissions: []models.Permission{
					{ID: uuid.New(), Name: "read"},
					{ID: uuid.New(), Name: "write"},
				},
			},
		},
	}

	mockCache.On("Get", mock.Anything, cacheKey).Return(invalidJSON, nil)
	repo.On("GetByIDWithPermissions", mock.Anything, userID).Return(user, nil)
	mockCache.On("Set", mock.Anything, cacheKey, mock.AnythingOfType("string"), 30*time.Minute).Return(nil)

	permissions, err := svc.GetUserPermissions(context.Background(), userID)

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"read", "write"}, permissions)

	repo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGetUserPermissions_CacheMiss(t *testing.T) {
	repo := &MockRepo{}
	mockCache := &cache.MockCache{}
	svc := NewAuthService(repo, mockCache)

	userID := uuid.New()
	cacheKey := "user:" + userID.String() + ":permissions"

	user := &models.User{
		ID: userID,
		Roles: []models.Role{
			{
				ID:   uuid.New(),
				Name: "admin",
				Permissions: []models.Permission{
					{ID: uuid.New(), Name: "read"},
					{ID: uuid.New(), Name: "write"},
				},
			},
		},
	}

	mockCache.On("Get", mock.Anything, cacheKey).Return("", cache.ErrCacheMiss)
	repo.On("GetByIDWithPermissions", mock.Anything, userID).Return(user, nil)
	mockCache.On("Set", mock.Anything, cacheKey, mock.AnythingOfType("string"), 30*time.Minute).Return(nil)

	permissions, err := svc.GetUserPermissions(context.Background(), userID)

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"read", "write"}, permissions)

	repo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGetUserPermissions_CacheError(t *testing.T) {
	repo := &MockRepo{}
	mockCache := &cache.MockCache{}
	svc := NewAuthService(repo, mockCache)

	userID := uuid.New()
	cacheKey := "user:" + userID.String() + ":permissions"

	user := &models.User{
		ID: userID,
		Roles: []models.Role{
			{
				ID:   uuid.New(),
				Name: "user",
				Permissions: []models.Permission{
					{ID: uuid.New(), Name: "read"},
				},
			},
		},
	}

	mockCache.On("Get", mock.Anything, cacheKey).Return("", errors.New("cache error"))
	repo.On("GetByIDWithPermissions", mock.Anything, userID).Return(user, nil)
	mockCache.On("Set", mock.Anything, cacheKey, mock.AnythingOfType("string"), 30*time.Minute).Return(nil)

	permissions, err := svc.GetUserPermissions(context.Background(), userID)

	assert.NoError(t, err)
	assert.Equal(t, []string{"read"}, permissions)

	repo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGetUserPermissions_RepoError(t *testing.T) {
	repo := &MockRepo{}
	mockCache := &cache.MockCache{}
	svc := NewAuthService(repo, mockCache)

	userID := uuid.New()
	cacheKey := "user:" + userID.String() + ":permissions"

	mockCache.On("Get", mock.Anything, cacheKey).Return("", cache.ErrCacheMiss)
	repo.On("GetByIDWithPermissions", mock.Anything, userID).Return(nil, gorm.ErrRecordNotFound)

	permissions, err := svc.GetUserPermissions(context.Background(), userID)

	assert.Nil(t, permissions)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	repo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
	mockCache.AssertNotCalled(t, "Set")
}

func TestGetUserPermissions_UserWithNoRoles(t *testing.T) {
	repo := &MockRepo{}
	mockCache := &cache.MockCache{}
	svc := NewAuthService(repo, mockCache)

	userID := uuid.New()
	cacheKey := "user:" + userID.String() + ":permissions"

	user := &models.User{
		ID:    userID,
		Roles: []models.Role{},
	}

	mockCache.On("Get", mock.Anything, cacheKey).Return("", cache.ErrCacheMiss)
	repo.On("GetByIDWithPermissions", mock.Anything, userID).Return(user, nil)
	mockCache.On("Set", mock.Anything, cacheKey, "[]", 30*time.Minute).Return(nil)

	permissions, err := svc.GetUserPermissions(context.Background(), userID)

	assert.NoError(t, err)
	assert.Empty(t, permissions)

	repo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGetUserPermissions_UserWithRolesButNoPermissions(t *testing.T) {
	repo := &MockRepo{}
	mockCache := &cache.MockCache{}
	svc := NewAuthService(repo, mockCache)

	userID := uuid.New()
	cacheKey := "user:" + userID.String() + ":permissions"

	user := &models.User{
		ID: userID,
		Roles: []models.Role{
			{
				ID:          uuid.New(),
				Name:        "empty_role",
				Permissions: []models.Permission{},
			},
		},
	}

	mockCache.On("Get", mock.Anything, cacheKey).Return("", cache.ErrCacheMiss)
	repo.On("GetByIDWithPermissions", mock.Anything, userID).Return(user, nil)
	mockCache.On("Set", mock.Anything, cacheKey, "[]", 30*time.Minute).Return(nil)

	permissions, err := svc.GetUserPermissions(context.Background(), userID)

	assert.NoError(t, err)
	assert.Empty(t, permissions)

	repo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGetUserPermissions_DuplicatePermissionsAcrossRoles(t *testing.T) {
	repo := &MockRepo{}
	mockCache := &cache.MockCache{}
	svc := NewAuthService(repo, mockCache)

	userID := uuid.New()
	cacheKey := "user:" + userID.String() + ":permissions"

	user := &models.User{
		ID: userID,
		Roles: []models.Role{
			{
				ID:   uuid.New(),
				Name: "admin",
				Permissions: []models.Permission{
					{ID: uuid.New(), Name: "read"},
					{ID: uuid.New(), Name: "write"},
				},
			},
			{
				ID:   uuid.New(),
				Name: "editor",
				Permissions: []models.Permission{
					{ID: uuid.New(), Name: "read"}, // Duplicate
					{ID: uuid.New(), Name: "edit"},
				},
			},
		},
	}

	mockCache.On("Get", mock.Anything, cacheKey).Return("", cache.ErrCacheMiss)
	repo.On("GetByIDWithPermissions", mock.Anything, userID).Return(user, nil)
	mockCache.On("Set", mock.Anything, cacheKey, mock.AnythingOfType("string"), 30*time.Minute).Return(nil)

	permissions, err := svc.GetUserPermissions(context.Background(), userID)

	assert.NoError(t, err)
	assert.Len(t, permissions, 3) // Should deduplicate 'read'
	assert.ElementsMatch(t, []string{"read", "write", "edit"}, permissions)

	repo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGetUserPermissions_CacheSetError(t *testing.T) {
	repo := &MockRepo{}
	mockCache := &cache.MockCache{}
	svc := NewAuthService(repo, mockCache)

	userID := uuid.New()
	cacheKey := "user:" + userID.String() + ":permissions"

	user := &models.User{
		ID: userID,
		Roles: []models.Role{
			{
				ID:   uuid.New(),
				Name: "user",
				Permissions: []models.Permission{
					{ID: uuid.New(), Name: "read"},
				},
			},
		},
	}

	mockCache.On("Get", mock.Anything, cacheKey).Return("", cache.ErrCacheMiss)
	repo.On("GetByIDWithPermissions", mock.Anything, userID).Return(user, nil)
	mockCache.On("Set", mock.Anything, cacheKey, mock.AnythingOfType("string"), 30*time.Minute).Return(errors.New("cache set error"))

	permissions, err := svc.GetUserPermissions(context.Background(), userID)

	assert.Equal(t, []string{"read"}, permissions)
	assert.EqualError(t, err, "cache set error") // Should return cache error

	repo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGetUserPermissions_MultipleRolesWithPermissions(t *testing.T) {
	repo := &MockRepo{}
	mockCache := &cache.MockCache{}
	svc := NewAuthService(repo, mockCache)

	userID := uuid.New()
	cacheKey := "user:" + userID.String() + ":permissions"

	user := &models.User{
		ID: userID,
		Roles: []models.Role{
			{
				ID:   uuid.New(),
				Name: "admin",
				Permissions: []models.Permission{
					{ID: uuid.New(), Name: "users.create"},
					{ID: uuid.New(), Name: "users.update"},
					{ID: uuid.New(), Name: "users.delete"},
				},
			},
			{
				ID:   uuid.New(),
				Name: "moderator",
				Permissions: []models.Permission{
					{ID: uuid.New(), Name: "posts.moderate"},
					{ID: uuid.New(), Name: "comments.moderate"},
				},
			},
			{
				ID:   uuid.New(),
				Name: "user",
				Permissions: []models.Permission{
					{ID: uuid.New(), Name: "posts.read"},
					{ID: uuid.New(), Name: "posts.create"},
				},
			},
		},
	}

	mockCache.On("Get", mock.Anything, cacheKey).Return("", cache.ErrCacheMiss)
	repo.On("GetByIDWithPermissions", mock.Anything, userID).Return(user, nil)
	mockCache.On("Set", mock.Anything, cacheKey, mock.AnythingOfType("string"), 30*time.Minute).Return(nil)

	permissions, err := svc.GetUserPermissions(context.Background(), userID)

	assert.NoError(t, err)
	assert.Len(t, permissions, 7)
	assert.ElementsMatch(t, []string{
		"users.create", "users.update", "users.delete",
		"posts.moderate", "comments.moderate",
		"posts.read", "posts.create",
	}, permissions)

	repo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}
