// internal/router/mounter.go
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/joefazee/neo/internal/deps"
)

// MountFunc represents a function that mounts routes for a module
type MountFunc func(*gin.RouterGroup, *deps.Container)

type Mounter struct {
	container *deps.Container
}

func NewMounter(container *deps.Container) *Mounter {
	return &Mounter{container: container}
}

// Public routes - no authentication required
func (m *Mounter) Public(engine *gin.Engine) *RouteGroup {
	group := engine.Group("/api/v1")
	return &RouteGroup{group: group, container: m.container}
}

// Authenticated routes - requires valid token
func (m *Mounter) Authenticated(engine *gin.Engine) *RouteGroup {
	group := engine.Group("/api/v1")
	// We'll add auth middleware here, but we need to do it without importing user package
	return &RouteGroup{group: group, container: m.container}
}

// Authorized routes - requires specific permission
func (m *Mounter) Authorized(engine *gin.Engine, permission string) *RouteGroup {
	group := engine.Group("/api/v1")
	// Middleware will be added dynamically
	return &RouteGroup{group: group, container: m.container, permission: permission}
}

type RouteGroup struct {
	group      *gin.RouterGroup
	container  *deps.Container
	permission string
}

// Mount provides a fluent interface for mounting modules
func (rg *RouteGroup) Mount(mountFunc MountFunc) *RouteGroup {
	mountFunc(rg.group, rg.container)
	return rg
}

// Group creates a sub-group for organizing routes
func (rg *RouteGroup) Group(path string) *RouteGroup {
	subGroup := rg.group.Group(path)
	return &RouteGroup{group: subGroup, container: rg.container, permission: rg.permission}
}

// WithAuth adds authentication middleware (called from main after user package is available)
func (rg *RouteGroup) WithAuth(authMiddleware gin.HandlerFunc) *RouteGroup {
	rg.group.Use(authMiddleware)
	return rg
}

// WithPermission adds permission middleware
func (rg *RouteGroup) WithPermission(permissionMiddleware gin.HandlerFunc) *RouteGroup {
	rg.group.Use(permissionMiddleware)
	return rg
}
