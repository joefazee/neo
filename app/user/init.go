package user

import (
	"log"

	"github.com/joefazee/neo/app/countries"

	"github.com/gin-gonic/gin"
	"github.com/joefazee/neo/internal/security"
	"gorm.io/gorm"
)

type Dependencies struct {
	DB     *gorm.DB
	Config *Config
}

func Init(r *gin.RouterGroup, deps Dependencies) {
	config := deps.Config
	if config == nil {
		config = GetDefaultConfig()
	}

	if err := config.Validate(); err != nil {
		panic("Invalid user configuration: " + err.Error())
	}

	repo := NewRepository(deps.DB)

	tokenMaker, err := security.NewPasetoMaker(config.SymmetricKey)
	if err != nil {
		log.Fatal("cannot create token maker: %w", err)
	}

	countryRepo := countries.NewRepository(deps.DB)
	srv := NewService(repo, tokenMaker)
	handler := NewHandler(srv, countryRepo)

	userGroup := r.Group("/users")
	userGroup.POST("/register", handler.Register)
	userGroup.POST("/login", handler.Login)
	userGroup.POST("/password-reset/request", handler.RequestPasswordReset)
	userGroup.POST("/password-reset/reset", handler.ResetPassword)
}
