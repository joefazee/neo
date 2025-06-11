// app/markets/init.go
package markets

import (
	"github.com/gin-gonic/gin"
	"github.com/joefazee/neo/app/categories"
	"github.com/joefazee/neo/app/countries"
	"github.com/joefazee/neo/internal/deps"
)

const (
	MarketRepoKey    = "market_repository"
	MarketServiceKey = "market_service"
)

func MountPublic(r *gin.RouterGroup, container *deps.Container) {
	handler := createHandler(container)

	marketsGroup := r.Group("/markets")
	marketsGroup.GET("", handler.GetMarkets)
	marketsGroup.GET("/:id", handler.GetMarketByID)
	marketsGroup.GET("/:id/prices", handler.GetMarketPrices)
	marketsGroup.GET("/category/:category_id", handler.GetMarketsByCategory)
}

func MountAuthenticated(r *gin.RouterGroup, container *deps.Container) {
	handler := createHandler(container)

	marketsGroup := r.Group("/markets")
	marketsGroup.POST("", handler.CreateMarket)
	marketsGroup.PUT("/:id", handler.UpdateMarket)
	marketsGroup.DELETE("/:id", handler.DeleteMarket)
	marketsGroup.GET("/my", handler.GetMyMarkets)
}

func MountAdmin(r *gin.RouterGroup, container *deps.Container) {
	handler := createHandler(container)

	marketsGroup := r.Group("/markets")
	marketsGroup.POST("/:id/resolve", handler.ResolveMarket)
	marketsGroup.POST("/:id/void", handler.VoidMarket)
}

func InitRepositories(container *deps.Container) {
	config := GetDefaultConfig()
	if err := config.Validate(); err != nil {
		panic("Invalid markets configuration: " + err.Error())
	}

	// Initialize engines
	pe := NewPricingEngine(config)
	se := NewSafeguardEngine(config)

	// Initialize repository
	repo := NewRepository(container.DB)
	container.RegisterRepository(MarketRepoKey, repo)

	// Initialize service
	service := NewService(repo, config, pe, se)
	container.RegisterService(MarketServiceKey, service)
}

func createHandler(container *deps.Container) *Handler {
	// Get dependencies from container
	service := container.GetService(MarketServiceKey).(Service)
	countryRepo := container.GetRepository(countries.CountryRepoKey).(countries.Repository)
	categoryRepo := container.GetRepository(categories.CategoryRepoKey).(categories.Repository)

	return NewHandler(service, countryRepo, categoryRepo, container.Sanitizer)
}
