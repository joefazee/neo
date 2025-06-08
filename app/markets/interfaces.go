package markets

import (
	"context"

	"github.com/google/uuid"
	"github.com/joefazee/neo/models"
)

// Repository defines the interface for market data access
type Repository interface {
	GetAll(ctx context.Context, filters *MarketFilters) ([]models.Market, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Market, error)
	GetByStatus(ctx context.Context, status models.MarketStatus) ([]models.Market, error)
	GetByCountryAndCategory(ctx context.Context, countryID, categoryID uuid.UUID) ([]models.Market, error)
	GetByCreator(ctx context.Context, creatorID uuid.UUID) ([]models.Market, error)
	GetExpiredMarkets(ctx context.Context) ([]models.Market, error)
	Create(ctx context.Context, market *models.Market) error
	Update(ctx context.Context, market *models.Market) error
	Delete(ctx context.Context, id uuid.UUID) error

	GetMarketOutcomes(ctx context.Context, marketID uuid.UUID) ([]models.MarketOutcome, error)
	CreateMarketOutcome(ctx context.Context, outcome *models.MarketOutcome) error
	UpdateMarketOutcome(ctx context.Context, outcome *models.MarketOutcome) error
	DeleteMarketOutcome(ctx context.Context, id uuid.UUID) error
}

// Service defines the interface for market business logic
type Service interface {
	GetMarkets(ctx context.Context, filters *MarketFilters) (*MarketListResponse, error)
	GetMarketByID(ctx context.Context, id uuid.UUID) (*MarketDetailResponse, error)
	GetMarketsByCategory(ctx context.Context, categoryID uuid.UUID) ([]MarketResponse, error)
	GetMyMarkets(ctx context.Context, userID uuid.UUID) ([]MarketResponse, error)
	CreateMarket(ctx context.Context, req *CreateMarketRequest) (*MarketDetailResponse, error)
	UpdateMarket(ctx context.Context, id uuid.UUID, req *UpdateMarketRequest) (*MarketDetailResponse, error)
	ResolveMarket(ctx context.Context, id uuid.UUID, req ResolveMarketRequest) (*MarketDetailResponse, error)
	VoidMarket(ctx context.Context, id uuid.UUID, reason string) error
	DeleteMarket(ctx context.Context, id uuid.UUID) error

	// Market outcomes
	AddMarketOutcome(ctx context.Context, marketID uuid.UUID, req CreateOutcomeRequest) (*OutcomeResponse, error)
	UpdateMarketOutcome(ctx context.Context, outcomeID uuid.UUID, req UpdateOutcomeRequest) (*OutcomeResponse, error)
	DeleteMarketOutcome(ctx context.Context, outcomeID uuid.UUID) error

	// Market engine
	CalculateCurrentPrices(ctx context.Context, marketID uuid.UUID) (map[string]PriceInfo, error)
	CheckSafeguards(ctx context.Context, marketID uuid.UUID) (*SafeguardStatus, error)
	ProcessExpiredMarkets(ctx context.Context) error
}

// PricingEngine defines the interface for market pricing calculations
type PricingEngine interface {
	CalculatePrice(totalPool, outcomePool float64) float64
	CalculatePriceImpact(currentPool, betAmount float64) float64
	CalculateContractsBought(betAmount, price float64) float64
	CalculatePayout(contracts, totalWinningContracts, prizePool float64) float64
}

// SafeguardEngine defines the interface for market safeguards
type SafeguardEngine interface {
	CheckQuorum(market *models.Market, outcomes []models.MarketOutcome) bool
	CheckImbalance(outcomes []models.MarketOutcome, threshold float64) bool
	ShouldTriggerHouseBot(market *models.Market, outcomes []models.MarketOutcome) bool
	CalculateHouseBotPosition(market *models.Market, outcomes []models.MarketOutcome) map[string]float64
}
