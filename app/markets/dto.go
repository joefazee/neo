package markets

import (
	"context"
	"strings"
	"time"

	"github.com/joefazee/neo/app/countries"

	"github.com/google/uuid"
	"github.com/joefazee/neo/app/categories"
	"github.com/joefazee/neo/internal/sanitizer"
	"github.com/joefazee/neo/internal/validator"
	"github.com/joefazee/neo/models"
	"github.com/shopspring/decimal"
)

// CreateMarketRequest represents the request to create a market
// @Description Request payload for creating a new prediction market
type CreateMarketRequest struct {
	// CountryID where the market will be available
	CountryID uuid.UUID `json:"country_id"`
	// CategoryID for market categorization
	CategoryID uuid.UUID `json:"category_id"`
	// Title Market title/question
	Title string `json:"title"`
	// Description Detailed market description
	Description string `json:"description"`
	// MarketType Type of market (binary or multi_outcome)
	MarketType string `json:"market_type"`
	// CloseTime When betting closes (RFC3339 format)
	CloseTime time.Time `json:"close_time"`
	// ResolutionDeadline Deadline for market resolution (RFC3339 format)
	ResolutionDeadline time.Time `json:"resolution_deadline"`
	// MinBetAmount Minimum bet amount (optional, uses country default if not provided)
	MinBetAmount decimal.Decimal `json:"min_bet_amount,omitempty"`
	// MaxBetAmount Maximum bet amount (optional)
	MaxBetAmount *decimal.Decimal `json:"max_bet_amount,omitempty"`
	// RakePercentage Platform fee percentage (optional, uses default if not provided)
	RakePercentage *decimal.Decimal `json:"rake_percentage,omitempty"`
	// CreatorRevenueShare Creator's share of the rake (optional, uses default if not provided)
	CreatorRevenueShare *decimal.Decimal `json:"creator_revenue_share,omitempty"`
	// Outcomes Market outcomes (minimum 2 required)
	Outcomes []CreateOutcomeRequest `json:"outcomes"`
	// SafeguardConfig Market safeguard settings (optional)
	SafeguardConfig *CreateSafeguardConfigRequest `json:"safeguard_config,omitempty"`
	// OracleConfig Oracle configuration for automated resolution (optional)
	OracleConfig *CreateOracleConfigRequest `json:"oracle_config,omitempty"`
	// Tags for market categorization (optional)
	Tags []string `json:"tags,omitempty"`
}

// CreateOutcomeRequest represents a market outcome in creation request
// @Description Individual outcome option for a prediction market
type CreateOutcomeRequest struct {
	OutcomeKey   string `json:"outcome_key"`
	OutcomeLabel string `json:"outcome_label"`
	SortOrder    int    `json:"sort_order,omitempty"`
}

// CreateSafeguardConfigRequest represents safeguard configuration
// @Description Configuration for market safeguards and risk management
type CreateSafeguardConfigRequest struct {
	// MinQuorumAmount Minimum total betting volume required to consider market valid
	MinQuorumAmount decimal.Decimal `json:"min_quorum_amount"`
	// MinOutcomes Minimum number of outcomes with bets to consider market valid
	MinOutcomes int `json:"min_outcomes"`
	// HouseBotEnabled Whether to enable house bot for market balancing
	HouseBotEnabled bool `json:"house_bot_enabled"`
	// HouseBotAmount Amount house bot will bet to balance market
	HouseBotAmount decimal.Decimal `json:"house_bot_amount"`
	// ImbalanceThreshold Threshold for market imbalance (0.8 = 80%)
	ImbalanceThreshold decimal.Decimal `json:"imbalance_threshold"`
	// VoidOnQuorumFail Whether to void market if quorum not met
	VoidOnQuorumFail bool `json:"void_on_quorum_fail"`
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

// Validate validates and sanitizes the create market request
//
// //nolint
func (r *CreateMarketRequest) Validate(ctx context.Context,
	v *validator.Validator,
	countryRepo countries.Repository,
	categoryRepo categories.Repository,
	s sanitizer.HTMLStripperer) bool {
	r.Title = s.StripHTML(strings.TrimSpace(r.Title))
	r.Description = s.StripHTML(strings.TrimSpace(r.Description))
	r.MarketType = s.StripHTML(strings.TrimSpace(strings.ToLower(r.MarketType)))

	v.Check(r.CountryID != uuid.Nil, "country_id", "Country ID is required")
	v.Check(r.CategoryID != uuid.Nil, "category_id", "Category ID is required")
	v.Check(validator.NotBlank(r.Title), "title", "Market title is required")
	v.Check(validator.MinRunes(r.Title, 10) && validator.MaxRunes(r.Title, 255), "title", "Market title must be between 10 and 255 characters")
	v.Check(validator.NotBlank(r.Description), "description", "Market description is required")
	v.Check(validator.MinRunes(r.Description, 50) &&
		validator.MaxRunes(r.Description, 2000),
		"description",
		"Market description must be between 50 and 2000 characters")
	v.Check(validator.In(r.MarketType,
		"binary",
		"multi_outcome"),
		"market_type",
		"Market type must be either 'binary' or 'multi_outcome'")

	now := time.Now()
	v.Check(
		r.CloseTime.After(now.Add(time.Hour)),
		"close_time",
		"Market close time must be at least 1 hour in the future",
	)
	v.Check(
		!r.ResolutionDeadline.Before(r.CloseTime.Add(time.Hour)),
		"resolution_deadline",
		"Resolution deadline must be at least 1 hour after close time",
	)
	maxDuration := 365 * 24 * time.Hour
	v.Check(r.CloseTime.Before(now.Add(maxDuration)), "close_time", "Market close time cannot be more than 1 year in the future")

	if r.CountryID != uuid.Nil {
		country, err := countryRepo.GetByID(ctx, r.CountryID)
		if err != nil {
			v.AddError("country_id", "Invalid country ID")
		} else {
			v.Check(country.IsActiveValue(), "country_id", "Selected country is not active")

			if r.MinBetAmount.IsZero() {
				r.MinBetAmount = country.GetMinBetAmount()
			}

			countryMinBet := country.GetMinBetAmount()
			countryMaxBet := country.GetMaxBetAmount()

			v.Check(r.MinBetAmount.GreaterThanOrEqual(countryMinBet), "min_bet_amount", "Minimum bet amount cannot be less than country minimum")
			if r.MaxBetAmount != nil {
				v.Check(r.MaxBetAmount.LessThanOrEqual(countryMaxBet), "max_bet_amount", "Maximum bet amount cannot exceed country maximum")
				v.Check(r.MaxBetAmount.GreaterThan(r.MinBetAmount), "max_bet_amount", "Maximum bet amount must be greater than minimum bet amount")
			}
		}
	}

	if r.CategoryID != uuid.Nil && r.CountryID != uuid.Nil {
		category, err := categoryRepo.GetByID(ctx, r.CategoryID)
		if err != nil {
			v.AddError("category_id", "Invalid category ID")
		} else {
			v.Check(category.IsActive, "category_id", "Selected category is not active")
			v.Check(category.CountryID == r.CountryID,
				"category_id",
				"Category does not belong to the selected country")
		}
	}

	if r.RakePercentage != nil {
		v.Check(r.RakePercentage.GreaterThanOrEqual(decimal.Zero) &&
			r.RakePercentage.LessThanOrEqual(decimal.NewFromFloat(0.2)),
			"rake_percentage",
			"Rake percentage must be between 0% and 20%")
	}

	if r.CreatorRevenueShare != nil {
		v.Check(r.CreatorRevenueShare.GreaterThanOrEqual(decimal.Zero) &&
			r.CreatorRevenueShare.LessThanOrEqual(decimal.NewFromInt(1)),
			"creator_revenue_share",
			"Creator revenue share must be between 0 and 1")
	}

	v.Check(len(r.Outcomes) >= 2, "outcomes", "Market must have at least 2 outcomes")
	v.Check(len(r.Outcomes) <= 10, "outcomes", "Market cannot have more than 10 outcomes")

	if len(r.Outcomes) > 0 {
		outcomeKeys := make(map[string]bool)
		outcomeLabels := make(map[string]bool)

		for i, outcome := range r.Outcomes {
			// Sanitize outcome fields
			r.Outcomes[i].OutcomeKey = s.StripHTML(strings.TrimSpace(strings.ToLower(outcome.OutcomeKey)))
			r.Outcomes[i].OutcomeLabel = s.StripHTML(strings.TrimSpace(outcome.OutcomeLabel))

			fieldPrefix := "outcomes[" + string(rune(i)) + "]"

			v.Check(validator.NotBlank(r.Outcomes[i].OutcomeKey), fieldPrefix+".outcome_key", "Outcome key is required")
			v.Check(validator.NotBlank(r.Outcomes[i].OutcomeLabel), fieldPrefix+".outcome_label", "Outcome label is required")
			v.Check(validator.MinRunes(r.Outcomes[i].OutcomeKey, 1) &&
				validator.MaxRunes(r.Outcomes[i].OutcomeKey, 50),
				fieldPrefix+".outcome_key",
				"Outcome key must be between 1 and 50 characters")
			v.Check(validator.MinRunes(r.Outcomes[i].OutcomeLabel, 1) &&
				validator.MaxRunes(r.Outcomes[i].OutcomeLabel, 100),
				fieldPrefix+".outcome_label",
				"Outcome label must be between 1 and 100 characters")

			// Check for duplicates
			if outcomeKeys[r.Outcomes[i].OutcomeKey] {
				v.AddError(fieldPrefix+".outcome_key", "Outcome keys must be unique")
			}
			if outcomeLabels[r.Outcomes[i].OutcomeLabel] {
				v.AddError(fieldPrefix+".outcome_label", "Outcome labels must be unique")
			}

			outcomeKeys[r.Outcomes[i].OutcomeKey] = true
			outcomeLabels[r.Outcomes[i].OutcomeLabel] = true
		}
	}

	if r.SafeguardConfig != nil {
		v.Check(r.SafeguardConfig.MinQuorumAmount.GreaterThan(decimal.Zero), "safeguard_config.min_quorum_amount", "Minimum quorum amount must be greater than zero")
		v.Check(r.SafeguardConfig.MinOutcomes >= 1, "safeguard_config.min_outcomes", "Minimum outcomes must be at least 1")
		v.Check(r.SafeguardConfig.MinOutcomes <= len(r.Outcomes), "safeguard_config.min_outcomes", "Minimum outcomes cannot exceed total outcomes")

		if r.SafeguardConfig.HouseBotEnabled {
			v.Check(r.SafeguardConfig.HouseBotAmount.GreaterThan(decimal.Zero),
				"safeguard_config.house_bot_amount",
				"House bot amount must be greater than zero when enabled")
		}

		v.Check(r.SafeguardConfig.ImbalanceThreshold.GreaterThan(decimal.NewFromFloat(0.5)) &&
			r.SafeguardConfig.ImbalanceThreshold.LessThanOrEqual(decimal.NewFromFloat(0.99)),
			"safeguard_config.imbalance_threshold",
			"Imbalance threshold must be between 0.5 and 0.99")
	}

	// Validate oracle config if provided
	if r.OracleConfig != nil {
		if validator.NotBlank(r.OracleConfig.Provider) {
			r.OracleConfig.Provider = s.StripHTML(strings.TrimSpace(r.OracleConfig.Provider))
			v.Check(validator.MaxRunes(r.OracleConfig.Provider, 50), "oracle_config.provider", "Oracle provider name too long")
		}

		if validator.NotBlank(r.OracleConfig.DataSource) {
			r.OracleConfig.DataSource = s.StripHTML(strings.TrimSpace(r.OracleConfig.DataSource))
			v.Check(validator.MaxRunes(r.OracleConfig.DataSource, 255), "oracle_config.data_source", "Oracle data source too long")
		}

		if validator.NotBlank(r.OracleConfig.ResolutionURL) {
			r.OracleConfig.ResolutionURL = s.StripHTML(strings.TrimSpace(r.OracleConfig.ResolutionURL))
			v.Check(validator.IsURL(r.OracleConfig.ResolutionURL), "oracle_config.resolution_url", "Invalid resolution URL")
		}

		if validator.NotBlank(r.OracleConfig.BackupProvider) {
			r.OracleConfig.BackupProvider = s.StripHTML(strings.TrimSpace(r.OracleConfig.BackupProvider))
			v.Check(validator.MaxRunes(r.OracleConfig.BackupProvider, 50), "oracle_config.backup_provider", "Backup provider name too long")
		}
	}

	// Validate and sanitize tags
	if len(r.Tags) > 0 {
		v.Check(len(r.Tags) <= 10, "tags", "Cannot have more than 10 tags")

		uniqueTags := make(map[string]bool)
		cleanTags := make([]string, 0, len(r.Tags))

		for _, tag := range r.Tags {
			cleanTag := s.StripHTML(strings.TrimSpace(strings.ToLower(tag)))
			if validator.NotBlank(cleanTag) {
				v.Check(validator.MaxRunes(cleanTag, 50), "tags", "Tag too long (max 50 characters)")
				if !uniqueTags[cleanTag] {
					cleanTags = append(cleanTags, cleanTag)
					uniqueTags[cleanTag] = true
				}
			}
		}
		r.Tags = cleanTags
	}

	return v.Valid()
}

// UpdateMarketRequest represents the request to update a market
// @Description Request payload for updating an existing market
type UpdateMarketRequest struct {
	Title              *string                       `json:"title,omitempty"`
	Description        *string                       `json:"description,omitempty"`
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
	WinningOutcome   string `json:"winning_outcome"`
	ResolutionSource string `json:"resolution_source"`
}

// UpdateOutcomeRequest represents the request to update a market outcome
// @Description Request payload for updating a market outcome
type UpdateOutcomeRequest struct {
	OutcomeLabel *string `json:"outcome_label,omitempty"`
	SortOrder    *int    `json:"sort_order,omitempty"`
}

// MarketFilters represents filters for market queries
// @Description Filters for searching and filtering markets
type MarketFilters struct {
	CountryID  uuid.UUID            `form:"country_id"`
	CategoryID uuid.UUID            `form:"category_id"`
	CreatorID  uuid.UUID            `form:"creator_id"`
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
	response.Outcomes = make([]OutcomeResponse, len(market.Outcomes))
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
