// app/prediction/init.go
package prediction

import (
	"github.com/gin-gonic/gin"
	"github.com/joefazee/neo/internal/deps"
)

const (
	RepoKey          = "prediction_repository"
	ServiceKey       = "prediction_service"
	BettingEngineKey = "betting_engine"
	RiskEngineKey    = "risk_engine"
)

// MountPublic mounts public prediction routes (quotes, price impact - read-only data)
func MountPublic(r *gin.RouterGroup, container *deps.Container) {
	handler := createHandler(container)

	// Public betting information endpoints
	bettingGroup := r.Group("/bets")
	bettingGroup.POST("/quote", handler.GetBetQuote)
	bettingGroup.GET("/markets/:market_id/outcomes/:outcome_id/price-impact", handler.GetPriceImpact)
}

// MountAuthenticated mounts authenticated prediction routes (user betting operations)
func MountAuthenticated(r *gin.RouterGroup, container *deps.Container) {
	handler := createHandler(container)

	bettingGroup := r.Group("/bets")

	// Core betting operations
	bettingGroup.POST("", handler.PlaceBet)
	bettingGroup.GET("", handler.GetMyBets)
	bettingGroup.GET("/:id", handler.GetBetByID)
	bettingGroup.POST("/:id/cancel", handler.CancelBet)

	// User portfolio and statistics
	bettingGroup.GET("/positions", handler.GetMyPositions)
	bettingGroup.GET("/portfolio", handler.GetMyPortfolio)
	bettingGroup.GET("/stats", handler.GetMyStats)
}

// InitRepositories initializes and registers repositories and services for this module
func InitRepositories(container *deps.Container) {
	// Get or create default config
	config := GetDefaultConfig()
	if err := config.Validate(); err != nil {
		panic("Invalid prediction configuration: " + err.Error())
	}

	// Initialize repository
	repo := NewRepository(container.DB)
	container.RegisterRepository(RepoKey, repo)

	// Initialize engines
	bettingEngine := NewBettingEngine(config)
	container.RegisterService(BettingEngineKey, bettingEngine)

	riskEngine := NewRiskEngine(config, repo)
	container.RegisterService(RiskEngineKey, riskEngine)

	// Initialize service
	service := NewService(container.DB, repo, config, bettingEngine, riskEngine)
	container.RegisterService(ServiceKey, service)
}

// createHandler creates a prediction handler with all dependencies
func createHandler(container *deps.Container) *Handler {
	service := container.GetService(ServiceKey).(Service)
	return NewHandler(service)
}
