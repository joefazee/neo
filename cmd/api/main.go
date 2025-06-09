package main

import (
	"fmt"
	"log"

	"github.com/joefazee/neo/internal/sanitizer"

	"github.com/joefazee/neo/internal/cache"
	"github.com/joefazee/neo/internal/security"

	"github.com/joefazee/neo/app/user"

	"github.com/joefazee/neo/app/markets"
	"github.com/joefazee/neo/app/prediction"

	"github.com/joefazee/neo/app"
	"github.com/joefazee/neo/app/api"
	"github.com/joefazee/neo/app/categories"
	"github.com/joefazee/neo/app/countries"
	"github.com/joefazee/neo/app/database"
	apiDoc "github.com/joefazee/neo/app/doc"
	_ "github.com/joefazee/neo/docs"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// @title Neo API
// @version 1.0
// @description Complete API for the Neo platform, providing endpoints for managing countries, categories, and more.
// @x-logo {"url": "https://go.dev/images/go-logo-white.svg", "altText": "Go API Logo"}
// @termsOfService https://argue-and-earn.com/terms

// @contact.name API Support Team
// @contact.url https://argue-and-earn.com/support
// @contact.email support@argue-and-earn.com

// @license.name MIT License
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @servers.url http://localhost:8080/
// @servers.description Local Development Server

// @servers.url https://staging.argue-and-earn.com/api/v1
// @servers.description Staging Server

// @servers.url https://argue-and-earn.com/api/v1
// @servers.description Production Server
func main() {
	cfg, err := app.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	db, err := database.New(&cfg.DB)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	HTMLSanitizer := sanitizer.NewHTMLStripper()
	cacheService := cache.NewCache[string](cache.MemoryBackend, nil)
	userRepo := user.NewRepository(db)
	authService := user.NewAuthService(userRepo, cacheService)

	tokenMaker, err := security.NewPasetoMaker(cfg.User.SymmetricKey)
	if err != nil {
		log.Fatal("cannot create token maker: %w", err)
	}

	r := gin.Default()

	apiV1 := r.Group("/api/v1")

	authGroup := apiV1.Group("/")
	authGroup.Use(user.AuthMiddleware(tokenMaker, authService))

	mountWithAuth(authGroup, db)
	mountWithoutAuth(apiV1, db, cfg, tokenMaker, HTMLSanitizer)
	apiDoc.Init(r)

	log.Printf("Starting Neo API server on %s:%s", cfg.AppHost, cfg.AppPort)
	if err := r.Run(fmt.Sprintf("%s:%s", cfg.AppHost, cfg.AppPort)); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func mountWithAuth(r *gin.RouterGroup, db *gorm.DB) {
	deps := struct {
		DB *gorm.DB
	}{
		DB: db,
	}
	countries.Init(r, deps)
	categories.Init(r, deps)
	markets.Init(r, markets.Dependencies{DB: db, Config: nil})
	prediction.Init(r, prediction.Dependencies{DB: db, Config: nil})
}

func mountWithoutAuth(r *gin.RouterGroup,
	db *gorm.DB,
	cfg *app.Config,
	maker security.Maker,
	sanitizer sanitizer.HTMLStripperer,
) {
	r.GET("/healthz", api.HealthCheck)
	user.Init(r, user.Dependencies{
		DB:         db,
		TokenMaker: maker,
		Config:     &cfg.User,
		Sanitizer:  sanitizer,
	})
}
