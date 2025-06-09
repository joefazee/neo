package markets

import (
	"github.com/gin-gonic/gin"
	"github.com/joefazee/neo/app/api"
	"gorm.io/gorm"
)

// Dependencies represents the dependencies needed for the markets module
type Dependencies struct {
	DB     *gorm.DB
	Config *Config
}

// Init initializes the markets module and mounts routes
func Init(r *gin.RouterGroup, deps Dependencies) {
	// Use default config if none provided
	config := deps.Config
	if config == nil {
		config = GetDefaultConfig()
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		panic("Invalid markets configuration: " + err.Error())
	}

	// Initialize engines
	pe := NewPricingEngine(config)
	se := NewSafeguardEngine(config)

	// Initialize repository
	repo := NewRepository(deps.DB)

	// Initialize service
	srvs := NewService(repo, config, pe, se)

	// Initialize handler
	handler := NewHandler(srvs)

	// Mount main market routes
	marketsGroup := r.Group("/markets")
	marketsGroup.GET("", handler.GetMarkets)
	marketsGroup.GET("/my", handler.GetMyMarkets)
	marketsGroup.GET("/:id", handler.GetMarketByID)
	marketsGroup.POST("", api.Can("market:create"), handler.CreateMarket)
	marketsGroup.PUT("/:id", handler.UpdateMarket)
	marketsGroup.DELETE("/:id", handler.DeleteMarket)

	// Market management routes
	marketsGroup.POST("/:id/resolve", handler.ResolveMarket)
	marketsGroup.POST("/:id/void", handler.VoidMarket)

	// Market outcomes routes
	marketsGroup.POST("/:id/outcomes", handler.AddMarketOutcome)

	// Market data routes
	marketsGroup.GET("/:id/prices", handler.GetMarketPrices)
	marketsGroup.GET("/:id/safeguards", handler.GetMarketSafeguards)

	// Mount outcome management routes
	outcomesGroup := r.Group("/outcomes")
	outcomesGroup.PUT("/:outcome_id", handler.UpdateMarketOutcome)
	outcomesGroup.DELETE("/:outcome_id", handler.DeleteMarketOutcome)

	marketsGroup.GET("/categories/:category_id/markets", handler.GetMarketsByCategory)
}
