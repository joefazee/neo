package prediction

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Dependencies represent the dependencies needed for the prediction module
type Dependencies struct {
	DB     *gorm.DB
	Config *Config
}

func Init(r *gin.RouterGroup, deps Dependencies) {
	if deps.Config == nil {
		deps.Config = GetDefaultConfig()
	}
	// Initialize repository
	repo := NewRepository(deps.DB)

	bengine := NewBettingEngine(deps.Config)
	rEngine := NewRiskEngine(deps.Config, repo)

	// Initialize service
	srvs := NewService(deps.DB, repo, deps.Config, bengine, rEngine)

	// Initialize handler
	handler := NewHandler(srvs)

	bettingGroup := r.Group("/bets")
	bettingGroup.POST("/", handler.PlaceBet)
	bettingGroup.GET("/", handler.GetMyBets)
	bettingGroup.POST("/quote", handler.GetBetQuote)
	bettingGroup.GET("/positions", handler.GetMyPositions)
	bettingGroup.GET("/portfolio", handler.GetMyPortfolio)
	bettingGroup.GET("/stats", handler.GetMyStats)
	bettingGroup.GET("/markets/{market_id}/outcomes/{outcome_id}/price-impact", handler.GetPriceImpact)
	bettingGroup.GET("/{id}", handler.GetBetByID)
	bettingGroup.POST("/{id}/cancel", handler.CancelBet)
}
