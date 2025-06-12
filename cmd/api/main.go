package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joefazee/neo/app/wallet"

	"github.com/gin-gonic/gin"
	"github.com/joefazee/neo/app"
	"github.com/joefazee/neo/app/api"
	"github.com/joefazee/neo/app/categories"
	"github.com/joefazee/neo/app/countries"
	"github.com/joefazee/neo/app/database"
	apiDoc "github.com/joefazee/neo/app/doc"
	"github.com/joefazee/neo/app/markets"
	"github.com/joefazee/neo/app/prediction"
	"github.com/joefazee/neo/app/user"
	_ "github.com/joefazee/neo/docs"
	"github.com/joefazee/neo/internal/cache"
	"github.com/joefazee/neo/internal/deps"
	"github.com/joefazee/neo/internal/logger"
	"github.com/joefazee/neo/internal/router"
	"github.com/joefazee/neo/internal/sanitizer"
	"github.com/joefazee/neo/internal/security"
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

	// Initialize core dependencies
	db, err := database.New(&cfg.DB)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	zeroLogger := logger.NewZeroLogger(os.Stdout, logger.LevelInfo, map[string]interface{}{
		"env":     cfg.Env,
		"service": "api",
	})

	htmlSanitizer := sanitizer.NewHTMLStripper()
	cacheService := cache.NewCache[string](cache.MemoryBackend, nil)

	tokenMaker, err := security.NewPasetoMaker(cfg.User.SymmetricKey)
	if err != nil {
		log.Fatal("cannot create token maker:", err)
	}

	container := deps.NewContainer(db, tokenMaker, htmlSanitizer, zeroLogger, cacheService)

	initializeRepositories(container)

	authService := user.NewAuthService(
		container.GetRepository(user.RepoKey).(user.Repository),
		cacheService,
	)
	container.RegisterService("auth_service", authService)

	r := gin.Default()
	mounter := router.NewMounter(container)

	mountRoutes(r, mounter, authService, tokenMaker)

	apiDoc.Init(r)

	log.Printf("Starting Neo API server on %s:%s", cfg.AppHost, cfg.AppPort)
	if err := r.Run(fmt.Sprintf("%s:%s", cfg.AppHost, cfg.AppPort)); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func initializeRepositories(container *deps.Container) {
	user.InitRepositories(container)
	countries.InitRepositories(container)
	categories.InitRepositories(container)
	markets.InitRepositories(container)
	prediction.InitRepositories(container)
	wallet.InitRepositories(container)
}

func mountRoutes(engine *gin.Engine, mounter *router.Mounter, authService user.AuthService, tokenMaker security.Maker) {
	mounter.Public(engine).
		Mount(func(r *gin.RouterGroup, _ *deps.Container) {
			r.GET("/healthz", api.HealthCheck)
		}).
		Mount(countries.MountPublic).
		Mount(categories.MountPublic).
		Mount(markets.MountPublic).
		Mount(user.MountPublic)

	mounter.Authenticated(engine).
		WithAuth(user.AuthMiddleware(tokenMaker, authService)).
		Mount(countries.MountAuthenticated).
		Mount(markets.MountAuthenticated).
		Mount(prediction.MountAuthenticated).
		Mount(wallet.MountAuthenticated).
		Mount(user.MountAuthenticated)

	mounter.Authorized(engine, "admin").
		WithAuth(user.AuthMiddleware(tokenMaker, authService)).
		WithPermission(api.Can("admin")).
		Mount(user.MountAdmin)

	mounter.Authorized(engine, "market:admin").
		WithAuth(user.AuthMiddleware(tokenMaker, authService)).
		WithPermission(api.Can("market:admin")).
		Mount(markets.MountAdmin)
}
