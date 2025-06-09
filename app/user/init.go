package user

import (
	"github.com/joefazee/neo/app/api"
	"github.com/joefazee/neo/app/countries"
	"github.com/joefazee/neo/internal/logger"
	"github.com/joefazee/neo/internal/sanitizer"

	"github.com/gin-gonic/gin"
	"github.com/joefazee/neo/internal/security"
	"gorm.io/gorm"
)

type Dependencies struct {
	DB          *gorm.DB
	Config      *Config
	TokenMaker  security.Maker
	Sanitizer   sanitizer.HTMLStripperer
	Logger      logger.Logger
	UserRepo    Repository
	CountryRepo countries.Repository
}

func Init(r *gin.RouterGroup, deps *Dependencies) {
	srv := NewService(deps.UserRepo, deps.TokenMaker)
	handler := NewHandler(srv, deps.CountryRepo, deps.Sanitizer, deps.Logger)

	userGroup := r.Group("/users")
	userGroup.POST("/register", handler.Register)
	userGroup.POST("/login", handler.Login)
	userGroup.POST("/password-reset/request", handler.RequestPasswordReset)
	userGroup.POST("/password-reset/reset", handler.ResetPassword)
}

func InitAdmin(r *gin.RouterGroup, deps *Dependencies) {
	as := NewAdminService(deps.UserRepo)
	adminHandler := NewAdminHandler(as, deps.Sanitizer, deps.Logger)

	// Admin routes
	adminGroup := r.Group("/admin/users")
	adminGroup.GET("", api.Can("admin:users:read"), adminHandler.GetUsers)
	adminGroup.PATCH("/:id/status", api.Can("admin:users:update_status"), adminHandler.UpdateUserStatus)
	adminGroup.POST("/:id/assign-role", api.Can("admin:users:assign_role"), adminHandler.AssignRoleToUser)
	adminGroup.POST("/bulk-assign-permissions", api.Can("admin:users:bulk_assign_permission"), adminHandler.BulkAssignPermissions)
}
