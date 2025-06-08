package prediction

import (
	"time"

	"github.com/google/uuid"
	"github.com/joefazee/neo/models"
	"github.com/shopspring/decimal"
)

// PlaceBetRequest represents the request to place a bet
// @Description Request payload for placing a bet on a market outcome
type PlaceBetRequest struct {
	MarketID       uuid.UUID       `json:"market_id" validate:"required"`
	OutcomeID      uuid.UUID       `json:"outcome_id" validate:"required"`
	Amount         decimal.Decimal `json:"amount" validate:"required,gt=0"`
	MaxSlippage    decimal.Decimal `json:"max_slippage,omitempty" validate:"omitempty,gte=0,lte=100"`
	ExpectedPrice  decimal.Decimal `json:"expected_price,omitempty" validate:"omitempty,gte=1,lte=99"`
	TimeoutSeconds int             `json:"timeout_seconds,omitempty" validate:"omitempty,min=5,max=300"`
}

// BetQuoteRequest represents the request for a bet quote
// @Description Request payload for getting a bet quote without placing the bet
type BetQuoteRequest struct {
	MarketID  uuid.UUID       `json:"market_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`  // Market ID
	OutcomeID uuid.UUID       `json:"outcome_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440001"` // Outcome ID
	Amount    decimal.Decimal `json:"amount" validate:"required,gt=0" example:"1000.00"`                             // Bet amount
}

// BetFilters represents filters for bet queries
// @Description Filters for searching and filtering user bets
type BetFilters struct {
	MarketID  *uuid.UUID        `form:"market_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	OutcomeID *uuid.UUID        `form:"outcome_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Status    *models.BetStatus `form:"status" example:"active"`
	DateFrom  *time.Time        `form:"date_from" example:"2024-01-01T00:00:00Z"`
	DateTo    *time.Time        `form:"date_to" example:"2024-12-31T23:59:59Z"`
	MinAmount *decimal.Decimal  `form:"min_amount" example:"100.00"`
	MaxAmount *decimal.Decimal  `form:"max_amount" example:"10000.00"`
	SortBy    string            `form:"sort_by" example:"created_at"`
	SortOrder string            `form:"sort_order" example:"desc"`
	Page      int               `form:"page" example:"1"`
	PerPage   int               `form:"per_page" example:"20"`
}

// BetResponse represents a bet in API responses
// @Description Bet information with current status and calculations
type BetResponse struct {
	ID               uuid.UUID        `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`         // Bet ID
	UserID           uuid.UUID        `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`    // User ID
	MarketID         uuid.UUID        `json:"market_id" example:"550e8400-e29b-41d4-a716-446655440002"`  // Market ID
	OutcomeID        uuid.UUID        `json:"outcome_id" example:"550e8400-e29b-41d4-a716-446655440003"` // Outcome ID
	Amount           decimal.Decimal  `json:"amount" example:"1000.00"`                                  // Bet amount
	ContractsBought  decimal.Decimal  `json:"contracts_bought" example:"20.00"`                          // Number of contracts bought
	PricePerContract decimal.Decimal  `json:"price_per_contract" example:"50.00"`                        // Price per contract
	TotalCost        decimal.Decimal  `json:"total_cost" example:"1000.00"`                              // Total cost including fees
	CurrentPrice     decimal.Decimal  `json:"current_price" example:"55.00"`                             // Current market price
	PotentialPayout  decimal.Decimal  `json:"potential_payout" example:"2000.00"`                        // Potential payout if wins
	ProfitLoss       decimal.Decimal  `json:"profit_loss" example:"100.00"`                              // Current P&L
	Status           string           `json:"status" example:"active"`                                   // Bet status
	PlacedAt         time.Time        `json:"placed_at" example:"2024-01-15T10:30:00Z"`                  // When bet was placed
	SettledAt        *time.Time       `json:"settled_at,omitempty" example:"2024-01-20T15:00:00Z"`       // When bet was settled
	SettlementAmount *decimal.Decimal `json:"settlement_amount,omitempty" example:"2000.00"`             // Settlement amount
	Market           *MarketSummary   `json:"market,omitempty"`                                          // Market summary
	Outcome          *OutcomeSummary  `json:"outcome,omitempty"`                                         // Outcome summary
}

// BetQuoteResponse represents a bet quote
// @Description Quote information for a potential bet
type BetQuoteResponse struct {
	MarketID          uuid.UUID       `json:"market_id" example:"550e8400-e29b-41d4-a716-446655440000"`  // Market ID
	OutcomeID         uuid.UUID       `json:"outcome_id" example:"550e8400-e29b-41d4-a716-446655440001"` // Outcome ID
	Amount            decimal.Decimal `json:"amount" example:"1000.00"`                                  // Bet amount
	CurrentPrice      decimal.Decimal `json:"current_price" example:"50.00"`                             // Current price
	EstimatedPrice    decimal.Decimal `json:"estimated_price" example:"52.00"`                           // Estimated price after bet
	PriceImpact       decimal.Decimal `json:"price_impact" example:"2.00"`                               // Price impact percentage
	ContractsBought   decimal.Decimal `json:"contracts_bought" example:"19.23"`                          // Contracts that would be bought
	PotentialPayout   decimal.Decimal `json:"potential_payout" example:"1923.00"`                        // Potential payout
	BreakevenPrice    decimal.Decimal `json:"breakeven_price" example:"52.00"`                           // Breakeven price
	MaxLoss           decimal.Decimal `json:"max_loss" example:"1000.00"`                                // Maximum possible loss
	EstimatedSlippage decimal.Decimal `json:"estimated_slippage" example:"2.00"`                         // Estimated slippage
	ValidUntil        time.Time       `json:"valid_until" example:"2024-01-15T10:35:00Z"`                // Quote expiry time
	Warnings          []string        `json:"warnings,omitempty" example:"[\"High slippage expected\"]"` // Warning messages
}

// PriceImpactResponse represents price impact analysis
// @Description Analysis of how a bet would affect market prices
type PriceImpactResponse struct {
	MarketID       uuid.UUID       `json:"market_id" example:"550e8400-e29b-41d4-a716-446655440000"`  // Market ID
	OutcomeID      uuid.UUID       `json:"outcome_id" example:"550e8400-e29b-41d4-a716-446655440001"` // Outcome ID
	BetAmount      decimal.Decimal `json:"bet_amount" example:"1000.00"`                              // Bet amount
	CurrentPrice   decimal.Decimal `json:"current_price" example:"50.00"`                             // Current price
	NewPrice       decimal.Decimal `json:"new_price" example:"52.00"`                                 // Price after bet
	PriceImpact    decimal.Decimal `json:"price_impact" example:"4.00"`                               // Price impact percentage
	LiquidityDepth decimal.Decimal `json:"liquidity_depth" example:"25000.00"`                        // Market liquidity
	ImpactCategory string          `json:"impact_category" example:"moderate"`                        // Impact category (low/moderate/high)
}

// PositionResponse represents a user's position in a market
// @Description User's current position in a specific market
type PositionResponse struct {
	MarketID          uuid.UUID       `json:"market_id" example:"550e8400-e29b-41d4-a716-446655440000"`  // Market ID
	MarketTitle       string          `json:"market_title" example:"Will OpenAI release GPT-5?"`         // Market title
	OutcomeID         uuid.UUID       `json:"outcome_id" example:"550e8400-e29b-41d4-a716-446655440001"` // Outcome ID
	OutcomeLabel      string          `json:"outcome_label" example:"Yes"`                               // Outcome label
	TotalInvested     decimal.Decimal `json:"total_invested" example:"5000.00"`                          // Total amount invested
	TotalContracts    decimal.Decimal `json:"total_contracts" example:"100.00"`                          // Total contracts owned
	AveragePrice      decimal.Decimal `json:"average_price" example:"50.00"`                             // Average price paid
	CurrentPrice      decimal.Decimal `json:"current_price" example:"60.00"`                             // Current market price
	CurrentValue      decimal.Decimal `json:"current_value" example:"6000.00"`                           // Current position value
	ProfitLoss        decimal.Decimal `json:"profit_loss" example:"1000.00"`                             // Unrealized P&L
	ProfitLossPercent decimal.Decimal `json:"profit_loss_percent" example:"20.00"`                       // P&L percentage
	BetCount          int             `json:"bet_count" example:"3"`                                     // Number of bets
	LastBetAt         time.Time       `json:"last_bet_at" example:"2024-01-15T10:30:00Z"`                // Last bet timestamp
}

// BetListResponse represents paginated bet list
// @Description Paginated list of user bets
type BetListResponse struct {
	Bets    []BetResponse `json:"bets"`     // List of bets
	Total   int64         `json:"total"`    // Total number of bets
	Page    int           `json:"page"`     // Current page
	PerPage int           `json:"per_page"` // Items per page
}

// PortfolioResponse represents user's betting portfolio
// @Description Complete user betting portfolio with summary statistics
type PortfolioResponse struct {
	UserID            uuid.UUID          `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"` // User ID
	TotalInvested     decimal.Decimal    `json:"total_invested" example:"50000.00"`                      // Total amount invested
	CurrentValue      decimal.Decimal    `json:"current_value" example:"55000.00"`                       // Current portfolio value
	TotalProfitLoss   decimal.Decimal    `json:"total_profit_loss" example:"5000.00"`                    // Total P&L
	ProfitLossPercent decimal.Decimal    `json:"profit_loss_percent" example:"10.00"`                    // P&L percentage
	ActivePositions   []PositionResponse `json:"active_positions"`                                       // Active positions
	TotalPositions    int                `json:"total_positions" example:"15"`                           // Total number of positions
	MarketsCount      int                `json:"markets_count" example:"8"`                              // Number of different markets
	WinRate           decimal.Decimal    `json:"win_rate" example:"65.5"`                                // Win rate percentage
	LastActivityAt    time.Time          `json:"last_activity_at" example:"2024-01-15T10:30:00Z"`        // Last betting activity
}

// BettingStatsResponse represents user betting statistics
// @Description Detailed betting statistics and performance metrics
type BettingStatsResponse struct {
	UserID         uuid.UUID       `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"` // User ID
	TotalBets      int             `json:"total_bets" example:"100"`                               // Total number of bets
	TotalAmount    decimal.Decimal `json:"total_amount" example:"100000.00"`                       // Total amount bet
	WonBets        int             `json:"won_bets" example:"65"`                                  // Number of winning bets
	LostBets       int             `json:"lost_bets" example:"30"`                                 // Number of losing bets
	PendingBets    int             `json:"pending_bets" example:"5"`                               // Number of pending bets
	WinRate        decimal.Decimal `json:"win_rate" example:"65.0"`                                // Win rate percentage
	AverageBetSize decimal.Decimal `json:"average_bet_size" example:"1000.00"`                     // Average bet size
	LargestWin     decimal.Decimal `json:"largest_win" example:"5000.00"`                          // Largest single win
	LargestLoss    decimal.Decimal `json:"largest_loss" example:"2000.00"`                         // Largest single loss
	TotalWinnings  decimal.Decimal `json:"total_winnings" example:"75000.00"`                      // Total winnings
	TotalLosses    decimal.Decimal `json:"total_losses" example:"30000.00"`                        // Total losses
	NetProfit      decimal.Decimal `json:"net_profit" example:"45000.00"`                          // Net profit
	ROI            decimal.Decimal `json:"roi" example:"45.0"`                                     // Return on investment
	BestCategory   string          `json:"best_category,omitempty" example:"AI"`                   // Best performing category
	WorstCategory  string          `json:"worst_category,omitempty" example:"Crypto"`              // Worst performing category
	FirstBetAt     time.Time       `json:"first_bet_at" example:"2024-01-01T10:30:00Z"`            // First bet timestamp
	LastBetAt      time.Time       `json:"last_bet_at" example:"2024-01-15T10:30:00Z"`             // Last bet timestamp
}

// MarketSummary represents a market summary for bet responses
// @Description Brief market information for bet context
type MarketSummary struct {
	ID        uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`               // Market ID
	Title     string    `json:"title" example:"Will OpenAI release GPT-5 before December 2024?"` // Market title
	Status    string    `json:"status" example:"open"`                                           // Market status
	CloseTime time.Time `json:"close_time" example:"2024-12-31T23:59:59Z"`                       // Market close time
}

// OutcomeSummary represents an outcome summary for bet responses
// @Description Brief outcome information for bet context
type OutcomeSummary struct {
	ID           uuid.UUID       `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"` // Outcome ID
	Key          string          `json:"key" example:"yes"`                                 // Outcome key
	Label        string          `json:"label" example:"Yes"`                               // Outcome label
	CurrentPrice decimal.Decimal `json:"current_price" example:"55.00"`                     // Current price
}

// ToBetResponse converts a models.Bet to BetResponse
func ToBetResponse(bet *models.Bet) *BetResponse {
	response := &BetResponse{
		ID:               bet.ID,
		UserID:           bet.UserID,
		MarketID:         bet.MarketID,
		OutcomeID:        bet.MarketOutcomeID,
		Amount:           bet.Amount,
		ContractsBought:  bet.ContractsBought,
		PricePerContract: bet.PricePerContract,
		TotalCost:        bet.TotalCost,
		Status:           string(bet.Status),
		PlacedAt:         bet.CreatedAt,
		SettledAt:        bet.SettledAt,
		SettlementAmount: bet.SettlementAmount,
	}

	// Add market and outcome info if available
	if bet.Market != nil {
		response.Market = &MarketSummary{
			ID:        bet.Market.ID,
			Title:     bet.Market.Title,
			Status:    string(bet.Market.Status),
			CloseTime: bet.Market.CloseTime,
		}
	}

	if bet.MarketOutcome != nil {
		response.Outcome = &OutcomeSummary{
			ID:    bet.MarketOutcome.ID,
			Key:   bet.MarketOutcome.OutcomeKey,
			Label: bet.MarketOutcome.OutcomeLabel,
		}
	}

	return response
}

// ToBetResponseList converts a slice of models.Bet to BetResponse
func ToBetResponseList(bets []models.Bet) []BetResponse {
	responses := make([]BetResponse, len(bets))
	for i := range bets {
		bet := bets[i]
		responses[i] = *ToBetResponse(&bet)
	}
	return responses
}
