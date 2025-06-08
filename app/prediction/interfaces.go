package prediction

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/joefazee/neo/models"
	"github.com/shopspring/decimal"
)

// Repository defines the interface for betting data access
type Repository interface {
	WithTx(tx *gorm.DB) Repository

	GetBetByID(ctx context.Context, id uuid.UUID) (*models.Bet, error)
	GetBetsByUser(ctx context.Context, userID uuid.UUID, filters *BetFilters) ([]models.Bet, int64, error)
	GetBetsByMarket(ctx context.Context, marketID uuid.UUID) ([]models.Bet, error)
	GetActiveBetsByUser(ctx context.Context, userID uuid.UUID) ([]models.Bet, error)
	CreateBet(ctx context.Context, bet *models.Bet) error
	UpdateBet(ctx context.Context, bet *models.Bet) error
	UpdateTransaction(ctx context.Context, transaction *models.Transaction) error

	// Position calculations
	GetUserPositionInMarket(ctx context.Context, userID, marketID uuid.UUID) (decimal.Decimal, error)
	GetUserDailyBetAmount(ctx context.Context, userID uuid.UUID, date time.Time) (decimal.Decimal, error)
	GetUserBetCount(ctx context.Context, userID uuid.UUID, since time.Time) (int, error)

	// Market data
	GetMarketWithOutcomes(ctx context.Context, marketID uuid.UUID) (*models.Market, error)
	GetMarketOutcome(ctx context.Context, outcomeID uuid.UUID) (*models.MarketOutcome, error)
	UpdateMarketOutcome(ctx context.Context, outcome *models.MarketOutcome) error
	UpdateMarket(ctx context.Context, market *models.Market) error

	// User wallet operations
	GetUserWallet(ctx context.Context, userID uuid.UUID, currencyCode string) (*models.Wallet, error)
	UpdateWallet(ctx context.Context, wallet *models.Wallet) error
	CreateTransaction(ctx context.Context, transaction *models.Transaction) error
}

// Service defines the interface for betting business logic
type Service interface {
	// Betting operations
	PlaceBet(ctx context.Context, userID uuid.UUID, req *PlaceBetRequest) (*BetResponse, error)
	CancelBet(ctx context.Context, userID, betID uuid.UUID) error
	GetBetByID(ctx context.Context, userID, betID uuid.UUID) (*BetResponse, error)
	GetUserBets(ctx context.Context, userID uuid.UUID, filters *BetFilters) (*BetListResponse, error)
	GetUserPositions(ctx context.Context, userID uuid.UUID) ([]PositionResponse, error)

	// Market analysis
	CalculateBetQuote(ctx context.Context, req BetQuoteRequest) (*BetQuoteResponse, error)
	GetMarketPriceImpact(ctx context.Context, marketID, outcomeID uuid.UUID, amount decimal.Decimal) (*PriceImpactResponse, error)

	// Portfolio management
	GetUserPortfolio(ctx context.Context, userID uuid.UUID) (*PortfolioResponse, error)
	GetUserBettingStats(ctx context.Context, userID uuid.UUID) (*BettingStatsResponse, error)
}

// BettingEngine defines the interface for core betting calculations
type BettingEngine interface {
	CalculateContractPrice(market *models.Market, outcome *models.MarketOutcome) decimal.Decimal
	CalculateContractsBought(betAmount, price decimal.Decimal) decimal.Decimal
	CalculatePriceImpact(currentPool, betAmount decimal.Decimal) decimal.Decimal
	CalculateSlippage(expectedPrice, actualPrice decimal.Decimal) decimal.Decimal
	ValidateSlippage(slippage, tolerance decimal.Decimal) error
	CalculateNewPrice(market *models.Market, outcome *models.MarketOutcome, betAmount decimal.Decimal) decimal.Decimal
	CalculateBreakevenPrice(betAmount, contracts decimal.Decimal) decimal.Decimal
	CalculatePotentialPayout(contracts decimal.Decimal, totalWinningContracts, prizePool decimal.Decimal) decimal.Decimal
	EstimateGasPrice(market *models.Market, outcome *models.MarketOutcome, betAmount decimal.Decimal) decimal.Decimal
	CalculateLiquidityScore(market *models.Market) decimal.Decimal
	CalculateImpliedProbability(price decimal.Decimal) decimal.Decimal
	CalculateExpectedValue(betAmount, price, trueProbability decimal.Decimal) decimal.Decimal
	CalculateOptimalBetSize(bankroll, price, trueProbability decimal.Decimal) decimal.Decimal
}

// RiskEngine defines the interface for betting risk management
type RiskEngine interface {
	CheckBettingLimits(userID uuid.UUID, amount decimal.Decimal, market *models.Market) error
	CheckPositionLimits(userID uuid.UUID, amount decimal.Decimal, market *models.Market) error
	CheckRateLimit(userID uuid.UUID) error
	CheckCooldown(userID uuid.UUID) error
	ValidateMarketForBetting(market *models.Market) error
	ValidateUserForBetting(user *models.User) error
	CheckWalletBalance(userID uuid.UUID, amount decimal.Decimal, currencyCode string) error
	AssessRiskScore(userID uuid.UUID, amount decimal.Decimal, market *models.Market) (decimal.Decimal, error)
}
