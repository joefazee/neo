package markets

import (
	"time"

	"github.com/google/uuid"
	"github.com/joefazee/neo/models"
	"github.com/shopspring/decimal"
)

// CreateMarketRequest represents the request to create a market
// @Description Request payload for creating a new prediction market
type CreateMarketRequest struct {

	// CountryID where the market will be available
	CountryID uuid.UUID `json:"country_id" binding:"required"`
	// CategoryID for market categorization
	CategoryID uuid.UUID `json:"category_id" binding:"required"`

	// Title Market title/question
	Title string `json:"title" binding:"required,min=10,max=255"`

	// @Description Detailed market description
	Description string `json:"description" binding:"required,min=50,max=2000"`

	// MarketType Type of market (binary or multi_outcome)
	MarketType string `json:"market_type" binding:"required,oneof=binary multi_outcome"`

	// CloseTime When betting closes
	CloseTime time.Time `json:"close_time" binding:"required"`

	// ResolutionDeadline Deadline for market resolution
	ResolutionDeadline time.Time `json:"resolution_deadline" binding:"required"`

	// MinBetAmount Minimum bet amount
	MinBetAmount decimal.Decimal `json:"min_bet_amount,omitempty"`

	// MaxBetAmount Maximum bet amount (optional)
	MaxBetAmount *decimal.Decimal `json:"max_bet_amount,omitempty"`

	// RakePercentage Platform fee percentage (optional)
	RakePercentage *decimal.Decimal `json:"rake_percentage,omitempty"`
	// CreatorRevenueShare
	CreatorRevenueShare *decimal.Decimal `json:"creator_revenue_share,omitempty"`

	// Outcomes represent the market outcomes
	Outcomes []CreateOutcomeRequest `json:"outcomes" binding:"required,min=2"`

	// SafeguardConfig represents the market safeguard configuration
	SafeguardConfig *CreateSafeguardConfigRequest `json:"safeguard_config,omitempty"`

	// OracleConfig represents the oracle configuration for automated resolution
	OracleConfig *CreateOracleConfigRequest `json:"oracle_config,omitempty"`

	// Tags for market categorization
	Tags []string `json:"tags,omitempty"`
}

// CreateOutcomeRequest represents a market outcome in creation request
// @Description Individual outcome option for a prediction market
type CreateOutcomeRequest struct {
	OutcomeKey   string `json:"outcome_key" binding:"required,min=1,max=50"`
	OutcomeLabel string `json:"outcome_label" binding:"required,min=1,max=100"`
	SortOrder    int    `json:"sort_order,omitempty"`
}

// CreateSafeguardConfigRequest represents safeguard configuration
// @Description Configuration for market safeguards and risk management
type CreateSafeguardConfigRequest struct {

	// MinQuorumAmount Minimum total betting volume required to consider market valid
	MinQuorumAmount decimal.Decimal `json:"min_quorum_amount" example:"5000.00"`

	// MinOutcomes Minimum number of outcomes with bets to consider market valid
	MinOutcomes int `json:"min_outcomes" example:"2"`

	// HouseBotEnabled Whether to enable house bot for market balancing
	HouseBotEnabled bool `json:"house_bot_enabled" example:"true"`

	// HouseBotAmount Amount house bot will bet to balance market
	HouseBotAmount decimal.Decimal `json:"house_bot_amount" example:"10000.00"`

	// ImbalanceThreshold Threshold for market imbalance (0.8 = 80%)
	ImbalanceThreshold decimal.Decimal `json:"imbalance_threshold" example:"0.80"`

	// VoidOnQuorumFail Whether to void market if quorum not met
	VoidOnQuorumFail bool `json:"void_on_quorum_fail" example:"true"`
}

// CreateOracleConfigRequest represents oracle configuration
// @Description Configuration for automated market resolution
type CreateOracleConfigRequest struct {
	Provider       string            `json:"provider,omitempty"`
	DataSource     string            `json:"data_source,omitempty"`
	ResolutionURL  string            `json:"resolution_url,omitempty"`
	Criteria       map[string]string `json:"criteria,omitempty"`
	AutoResolve    bool              `json:"auto_resolve"`
	BackupProvider string            `json:"backup_provider,omitempty"`
}

// UpdateMarketRequest represents the request to update a market
// @Description Request payload for updating an existing market
type UpdateMarketRequest struct {
	Title              *string                       `json:"title,omitempty" binding:"omitempty,min=10,max=255"`
	Description        *string                       `json:"description,omitempty" binding:"omitempty,min=50,max=2000"`
	CloseTime          *time.Time                    `json:"close_time,omitempty"`
	ResolutionDeadline *time.Time                    `json:"resolution_deadline,omitempty"`
	MinBetAmount       *decimal.Decimal              `json:"min_bet_amount,omitempty"`
	MaxBetAmount       *decimal.Decimal              `json:"max_bet_amount,omitempty"`
	SafeguardConfig    *CreateSafeguardConfigRequest `json:"safeguard_config,omitempty"`
	OracleConfig       *CreateOracleConfigRequest    `json:"oracle_config,omitempty"`
	Tags               []string                      `json:"tags,omitempty"`
}

// ResolveMarketRequest represents the request to resolve a market
// @Description Request payload for resolving a prediction market
type ResolveMarketRequest struct {
	WinningOutcome   string `json:"winning_outcome" binding:"required"`
	ResolutionSource string `json:"resolution_source" binding:"required"`
}

// UpdateOutcomeRequest represents the request to update a market outcome
// @Description Request payload for updating a market outcome
type UpdateOutcomeRequest struct {
	OutcomeLabel *string `json:"outcome_label,omitempty" binding:"omitempty,min=1,max=100"`
	SortOrder    *int    `json:"sort_order,omitempty"`
}

// MarketFilters represents filters for market queries
// @Description Filters for searching and filtering markets
type MarketFilters struct {
	CountryID  *uuid.UUID           `form:"country_id"`
	CategoryID *uuid.UUID           `form:"category_id"`
	CreatorID  *uuid.UUID           `form:"creator_id"`
	Status     *models.MarketStatus `form:"status"`
	MarketType *models.MarketType   `form:"market_type"`
	Tags       []string             `form:"tags"`
	Search     string               `form:"search"`
	SortBy     string               `form:"sort_by"`
	SortOrder  string               `form:"sort_order"`
	Page       int                  `form:"page"`
	PerPage    int                  `form:"per_page"`
}

// MarketResponse represents a market in list view
// @Description Market information for list display
type MarketResponse struct {
	ID                 uuid.UUID       `json:"id"`
	Title              string          `json:"title"`
	Description        string          `json:"description"`
	MarketType         string          `json:"market_type"`
	Status             string          `json:"status"`
	CloseTime          time.Time       `json:"close_time"`
	ResolutionDeadline time.Time       `json:"resolution_deadline"`
	TotalPoolAmount    decimal.Decimal `json:"total_pool_amount"`
	CreatedAt          time.Time       `json:"created_at"`
	OutcomeCount       int             `json:"outcome_count"`
	Tags               []string        `json:"tags"`
}

// MarketDetailResponse represents detailed market information
// @Description Detailed market information including outcomes and pricing
type MarketDetailResponse struct {
	ID                  uuid.UUID               `json:"id"`
	CountryID           uuid.UUID               `json:"country_id"`
	CategoryID          uuid.UUID               `json:"category_id"`
	CreatorID           *uuid.UUID              `json:"creator_id"`
	Title               string                  `json:"title"`
	Description         string                  `json:"description"`
	MarketType          string                  `json:"market_type"`
	Status              string                  `json:"status"`
	CloseTime           time.Time               `json:"close_time"`
	ResolutionDeadline  time.Time               `json:"resolution_deadline"`
	ResolvedAt          *time.Time              `json:"resolved_at,omitempty"`
	ResolvedOutcome     *string                 `json:"resolved_outcome,omitempty"`
	ResolutionSource    *string                 `json:"resolution_source,omitempty"`
	MinBetAmount        decimal.Decimal         `json:"min_bet_amount"`
	MaxBetAmount        *decimal.Decimal        `json:"max_bet_amount,omitempty"`
	TotalPoolAmount     decimal.Decimal         `json:"total_pool_amount"`
	RakePercentage      decimal.Decimal         `json:"rake_percentage"`
	CreatorRevenueShare decimal.Decimal         `json:"creator_revenue_share"`
	Outcomes            []OutcomeResponse       `json:"outcomes"`
	SafeguardConfig     SafeguardConfigResponse `json:"safeguard_config"`
	OracleConfig        OracleConfigResponse    `json:"oracle_config"`
	Metadata            MarketMetadataResponse  `json:"metadata"`
	CreatedAt           time.Time               `json:"created_at"`
	UpdatedAt           time.Time               `json:"updated_at"`
}

// OutcomeResponse represents a market outcome with current pricing
// @Description Market outcome with current price and betting information
type OutcomeResponse struct {
	ID               uuid.UUID       `json:"id"`
	OutcomeKey       string          `json:"outcome_key"`
	OutcomeLabel     string          `json:"outcome_label"`
	SortOrder        int             `json:"sort_order"`
	PoolAmount       decimal.Decimal `json:"pool_amount"`
	CurrentPrice     decimal.Decimal `json:"current_price"`
	IsWinningOutcome *bool           `json:"is_winning_outcome,omitempty"`
	BetCount         int             `json:"bet_count"`
	UniqueBackers    int             `json:"unique_backers"`
}

// MarketListResponse represents paginated market list
// @Description Paginated list of markets with metadata
type MarketListResponse struct {
	Markets []MarketResponse `json:"markets"`
	Total   int64            `json:"total"`
	Page    int              `json:"page"`
	PerPage int              `json:"per_page"`
}

// SafeguardConfigResponse represents safeguard configuration
// @Description Market safeguard configuration details
type SafeguardConfigResponse struct {
	MinQuorumAmount    decimal.Decimal `json:"min_quorum_amount"`
	MinOutcomes        int             `json:"min_outcomes"`
	HouseBotEnabled    bool            `json:"house_bot_enabled"`
	HouseBotAmount     decimal.Decimal `json:"house_bot_amount"`
	ImbalanceThreshold decimal.Decimal `json:"imbalance_threshold"`
	VoidOnQuorumFail   bool            `json:"void_on_quorum_fail"`
}

// OracleConfigResponse represents oracle configuration
// @Description Oracle configuration for automated resolution
type OracleConfigResponse struct {
	Provider       string            `json:"provider,omitempty"`
	DataSource     string            `json:"data_source,omitempty"`
	ResolutionURL  string            `json:"resolution_url,omitempty"`
	Criteria       map[string]string `json:"criteria,omitempty"`
	AutoResolve    bool              `json:"auto_resolve"`
	BackupProvider string            `json:"backup_provider,omitempty"`
}

// MarketMetadataResponse represents market metadata
// @Description Additional market metadata and statistics
type MarketMetadataResponse struct {
	Tags          []string   `json:"tags"`
	ImageURL      string     `json:"image_url,omitempty"`
	SourceURL     string     `json:"source_url,omitempty"`
	FeaturedUntil *time.Time `json:"featured_until,omitempty"`
	ViewCount     int64      `json:"view_count"`
}

// PriceInfo represents pricing information for an outcome
// @Description Current pricing information for a market outcome
type PriceInfo struct {
	CurrentPrice   decimal.Decimal `json:"current_price"`
	PriceChange24h decimal.Decimal `json:"price_change_24h"`
	Volume24h      decimal.Decimal `json:"volume_24h"`
	LastTradePrice decimal.Decimal `json:"last_trade_price"`
}

// SafeguardStatus represents the current safeguard status
// @Description Current status of market safeguards
type SafeguardStatus struct {
	QuorumMet         bool            `json:"quorum_met"`
	IsBalanced        bool            `json:"is_balanced"`
	HouseBotActive    bool            `json:"house_bot_active"`
	VoidRisk          bool            `json:"void_risk"`
	CurrentQuorum     decimal.Decimal `json:"current_quorum"`
	ImbalanceRatio    decimal.Decimal `json:"imbalance_ratio"`
	RecommendedAction string          `json:"recommended_action"`
}

// ToMarketResponse converts a models.Market to MarketResponse
func ToMarketResponse(market *models.Market) *MarketResponse {
	return &MarketResponse{
		ID:                 market.ID,
		Title:              market.Title,
		Description:        market.Description,
		MarketType:         string(market.MarketType),
		Status:             string(market.Status),
		CloseTime:          market.CloseTime,
		ResolutionDeadline: market.ResolutionDeadline,
		TotalPoolAmount:    market.TotalPoolAmount,
		CreatedAt:          market.CreatedAt,
		OutcomeCount:       len(market.Outcomes),
		Tags:               market.Metadata.Tags,
	}
}

// ToMarketDetailResponse converts a models.Market to MarketDetailResponse
func ToMarketDetailResponse(market *models.Market) *MarketDetailResponse {
	response := &MarketDetailResponse{
		ID:                  market.ID,
		CountryID:           market.CountryID,
		CategoryID:          market.CategoryID,
		CreatorID:           market.CreatorID,
		Title:               market.Title,
		Description:         market.Description,
		MarketType:          string(market.MarketType),
		Status:              string(market.Status),
		CloseTime:           market.CloseTime,
		ResolutionDeadline:  market.ResolutionDeadline,
		ResolvedAt:          market.ResolvedAt,
		MinBetAmount:        market.MinBetAmount,
		MaxBetAmount:        market.MaxBetAmount,
		TotalPoolAmount:     market.TotalPoolAmount,
		RakePercentage:      market.RakePercentage,
		CreatorRevenueShare: market.CreatorRevenueShare,
		CreatedAt:           market.CreatedAt,
		UpdatedAt:           market.UpdatedAt,
	}

	if market.ResolvedOutcome != "" {
		response.ResolvedOutcome = &market.ResolvedOutcome
	}
	if market.ResolutionSource != "" {
		response.ResolutionSource = &market.ResolutionSource
	}

	// Convert outcomes
	for i := range market.Outcomes {
		response.Outcomes[i] = *ToOutcomeResponse(&market.Outcomes[i])
	}

	// Convert safeguard config
	response.SafeguardConfig = SafeguardConfigResponse{
		MinQuorumAmount:    market.SafeguardConfig.MinQuorumAmount,
		MinOutcomes:        market.SafeguardConfig.MinOutcomes,
		HouseBotEnabled:    market.SafeguardConfig.HouseBotEnabled,
		HouseBotAmount:     market.SafeguardConfig.HouseBotAmount,
		ImbalanceThreshold: market.SafeguardConfig.ImbalanceThreshold,
		VoidOnQuorumFail:   market.SafeguardConfig.VoidOnQuorumFail,
	}

	// Convert oracle config
	response.OracleConfig = OracleConfigResponse{
		Provider:       market.OracleConfig.Provider,
		DataSource:     market.OracleConfig.DataSource,
		ResolutionURL:  market.OracleConfig.ResolutionURL,
		Criteria:       market.OracleConfig.Criteria,
		AutoResolve:    market.OracleConfig.AutoResolve,
		BackupProvider: market.OracleConfig.BackupProvider,
	}

	// Convert metadata
	response.Metadata = MarketMetadataResponse{
		Tags:          market.Metadata.Tags,
		ImageURL:      market.Metadata.ImageURL,
		SourceURL:     market.Metadata.SourceURL,
		FeaturedUntil: market.Metadata.FeaturedUntil,
		ViewCount:     market.Metadata.ViewCount,
	}

	return response
}

// ToOutcomeResponse converts a models.MarketOutcome to OutcomeResponse
func ToOutcomeResponse(outcome *models.MarketOutcome) *OutcomeResponse {
	return &OutcomeResponse{
		ID:               outcome.ID,
		OutcomeKey:       outcome.OutcomeKey,
		OutcomeLabel:     outcome.OutcomeLabel,
		SortOrder:        outcome.SortOrder,
		PoolAmount:       outcome.PoolAmount,
		IsWinningOutcome: outcome.IsWinningOutcome,
	}
}

// ToMarketResponseList converts a slice of models.Market to MarketResponse
func ToMarketResponseList(markets []models.Market) []MarketResponse {
	responses := make([]MarketResponse, len(markets))
	for i := range markets {
		market := markets[i]
		responses[i] = *ToMarketResponse(&market)
	}
	return responses
}
