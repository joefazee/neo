package user

import (
	"github.com/joefazee/neo/app/countries"
	"github.com/joefazee/neo/internal/sanitizer"

	"github.com/gin-gonic/gin"
	"github.com/joefazee/neo/internal/security"
	"gorm.io/gorm"
)

type Dependencies struct {
	DB         *gorm.DB
	Config     *Config
	TokenMaker security.Maker
	Sanitizer  sanitizer.HTMLStripperer
}

func Init(r *gin.RouterGroup, deps Dependencies) {
	repo := NewRepository(deps.DB)

	countryRepo := countries.NewRepository(deps.DB)
	srv := NewService(repo, deps.TokenMaker)
	handler := NewHandler(srv, countryRepo, deps.Sanitizer)

	userGroup := r.Group("/users")
	userGroup.POST("/register", handler.Register)
	userGroup.POST("/login", handler.Login)
	userGroup.POST("/password-reset/request", handler.RequestPasswordReset)
	userGroup.POST("/password-reset/reset", handler.ResetPassword)
}
