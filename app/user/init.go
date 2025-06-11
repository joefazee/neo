package user

import (
	"github.com/gin-gonic/gin"
	"github.com/joefazee/neo/app/api"
	"github.com/joefazee/neo/app/countries"
	"github.com/joefazee/neo/internal/deps"
)

const (
	RepoKey         = "user_repository"
	ServiceKey      = "user_service"
	AdminServiceKey = "admin_service"
	AuthServiceKey  = "auth_service"
)

// MountPublic mounts public user routes (registration, login, password reset)
func MountPublic(r *gin.RouterGroup, container *deps.Container) {
	handler := createHandler(container)

	userGroup := r.Group("/users")
	userGroup.POST("/register", handler.Register)
	userGroup.POST("/login", handler.Login)
	userGroup.POST("/password-reset/request", handler.RequestPasswordReset)
	userGroup.POST("/password-reset/reset", handler.ResetPassword)
}

// MountAuthenticated mounts authenticated user routes (profile management, etc.)
func MountAuthenticated(r *gin.RouterGroup, container *deps.Container) {
	handler := createHandler(container)

	userGroup := r.Group("/users")
	userGroup.GET("/profile", handler.GetProfile)
}

func MountAdmin(r *gin.RouterGroup, container *deps.Container) {
	adminHandler := createAdminHandler(container)

	// User management routes
	adminGroup := r.Group("/admin/users")
	adminGroup.GET("", api.Can("admin:users:read"), adminHandler.GetUsers)
	adminGroup.GET("/:id", api.Can("admin:users:read"), adminHandler.GetUserByID) // New route
	adminGroup.PATCH("/:id/status", api.Can("admin:users:update_status"), adminHandler.UpdateUserStatus)
	adminGroup.POST("/:id/assign-role", api.Can("admin:users:assign_role"), adminHandler.AssignRoleToUser)
	adminGroup.DELETE("/:id/roles/:role_id", api.Can("admin:users:remove_role"), adminHandler.RemoveRoleFromUser) // New route
	adminGroup.POST("/bulk-assign-permissions", api.Can("admin:users:bulk_assign_permission"), adminHandler.BulkAssignPermissions)

	// Permission management routes
	permissionGroup := r.Group("/admin/permissions")
	permissionGroup.POST("", api.Can("admin:permissions:create"), adminHandler.CreatePermission)

	// Role management routes
	roleGroup := r.Group("/admin/roles")
	roleGroup.POST("", api.Can("admin:roles:create"), adminHandler.CreateRole)
	roleGroup.PUT("/:id", api.Can("admin:roles:update"), adminHandler.UpdateRole)
	roleGroup.POST("/:id/permissions", api.Can("admin:roles:assign_permissions"), adminHandler.AssignPermissionsToRole)
	roleGroup.DELETE("/:id/permissions", api.Can("admin:roles:remove_permissions"), adminHandler.RemovePermissionsFromRole)
}

// InitRepositories initializes and registers repositories and services for this module
func InitRepositories(container *deps.Container) {
	// Initialize repository
	userRepo := NewRepository(container.DB)
	container.RegisterRepository(RepoKey, userRepo)

	// Initialize user service
	userService := NewService(userRepo, container.TokenMaker)
	container.RegisterService(ServiceKey, userService)

	// Initialize admin service
	adminService := NewAdminService(userRepo)
	container.RegisterService(AdminServiceKey, adminService)

	// Auth service will be initialized in main.go since it needs cache
}

// createHandler creates a user handler with all dependencies
func createHandler(container *deps.Container) *Handler {
	userService := container.GetService(ServiceKey).(Service)
	countryRepo := container.GetRepository(countries.CountryRepoKey).(countries.Repository)

	return NewHandler(userService, countryRepo, container.Sanitizer, container.Logger)
}

// createAdminHandler creates an admin handler with all dependencies
func createAdminHandler(container *deps.Container) *AdminHandler {
	adminService := container.GetService(AdminServiceKey).(AdminService)

	return NewAdminHandler(adminService, container.Sanitizer, container.Logger)
}
