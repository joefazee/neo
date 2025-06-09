package user

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/joefazee/neo/internal/cache"
)

type AuthService interface {
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error)
}

type authService struct {
	repo  Repository
	cache cache.Cache[string]
}

func NewAuthService(repo Repository, cache cache.Cache[string]) AuthService {
	return &authService{repo: repo, cache: cache}
}

func (s *authService) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	cacheKey := fmt.Sprintf("user:%s:permissions", userID)

	cachedPermissions, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cachedPermissions != "" {
		var permissions []string
		if err := json.Unmarshal([]byte(cachedPermissions), &permissions); err == nil {
			return permissions, nil
		}
	}

	user, err := s.repo.GetByIDWithPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	permissionsMap := make(map[string]struct{})
	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			permissionsMap[perm.Name] = struct{}{}
		}
	}

	permissions := make([]string, 0, len(permissionsMap))
	for perm := range permissionsMap {
		permissions = append(permissions, perm)
	}

	permissionsJSON, err := json.Marshal(permissions)
	if err == nil {
		err = s.cache.Set(ctx, cacheKey, string(permissionsJSON), 30*time.Minute)
	}

	return permissions, err
}
