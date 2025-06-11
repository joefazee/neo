package user

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joefazee/neo/app/countries"
	"github.com/joefazee/neo/internal/cache"
	"github.com/joefazee/neo/internal/deps"
	"github.com/joefazee/neo/internal/logger"
	"github.com/joefazee/neo/internal/security"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestMountPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	container := createTestContainer()

	MountPublic(router.Group("/api/v1"), container)

	routes := router.Routes()
	assertRouteExists(t, routes, "POST", "/api/v1/users/register")
	assertRouteExists(t, routes, "POST", "/api/v1/users/login")
	assertRouteExists(t, routes, "POST", "/api/v1/users/password-reset/request")
	assertRouteExists(t, routes, "POST", "/api/v1/users/password-reset/reset")
}

func TestMountAuthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	container := createTestContainer()

	MountAuthenticated(router.Group("/api/v1"), container)

	routes := router.Routes()
	assertRouteExists(t, routes, "GET", "/api/v1/users/profile")
}

func TestMountAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	container := createTestContainer()

	MountAdmin(router.Group("/api/v1"), container)

	routes := router.Routes()

	assertRouteExists(t, routes, "GET", "/api/v1/admin/users")
	assertRouteExists(t, routes, "GET", "/api/v1/admin/users/:id")
	assertRouteExists(t, routes, "PATCH", "/api/v1/admin/users/:id/status")
	assertRouteExists(t, routes, "POST", "/api/v1/admin/users/:id/assign-role")
	assertRouteExists(t, routes, "DELETE", "/api/v1/admin/users/:id/roles/:role_id")
	assertRouteExists(t, routes, "POST", "/api/v1/admin/users/bulk-assign-permissions")

	assertRouteExists(t, routes, "POST", "/api/v1/admin/permissions")

	assertRouteExists(t, routes, "POST", "/api/v1/admin/roles")
	assertRouteExists(t, routes, "PUT", "/api/v1/admin/roles/:id")
	assertRouteExists(t, routes, "POST", "/api/v1/admin/roles/:id/permissions")
	assertRouteExists(t, routes, "DELETE", "/api/v1/admin/roles/:id/permissions")
}

func TestInitRepositories(t *testing.T) {
	container := createTestContainer()

	InitRepositories(container)

	userRepo := container.GetRepository(RepoKey)
	assert.NotNil(t, userRepo)
	assert.Implements(t, (*Repository)(nil), userRepo)

	userService := container.GetService(ServiceKey)
	assert.NotNil(t, userService)
	assert.Implements(t, (*Service)(nil), userService)

	adminService := container.GetService(AdminServiceKey)
	assert.NotNil(t, adminService)
	assert.Implements(t, (*AdminService)(nil), adminService)
}

func createTestContainer() *deps.Container {
	container := deps.NewContainer(
		&gorm.DB{},
		&security.MockMaker{},
		&MockSanitizer{},
		logger.NewNullLogger(),
		&cache.MockCache{},
	)

	container.RegisterRepository(countries.CountryRepoKey, &MockCountryRepo{})
	container.RegisterService(ServiceKey, &MockService{})
	container.RegisterService(AdminServiceKey, &MockAdminService{})

	return container
}

func assertRouteExists(t *testing.T, routes []gin.RouteInfo, method, path string) {
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return
		}
	}
	t.Errorf("Route %s %s not found", method, path)
}
