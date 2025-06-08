package markets

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/joefazee/neo/models"
)

// service implements the Service interface
type service struct {
	repo            Repository
	config          *Config
	pricingEngine   PricingEngine
	safeguardEngine SafeguardEngine
}

// NewService creates a new market service
func NewService(repo Repository, config *Config, pricingEngine PricingEngine, safeguardEngine SafeguardEngine) Service {
	return &service{
		repo:            repo,
		config:          config,
		pricingEngine:   pricingEngine,
		safeguardEngine: safeguardEngine,
	}
}

// GetMarkets returns paginated markets with filters
func (s *service) GetMarkets(ctx context.Context, filters *MarketFilters) (*MarketListResponse, error) {
	markets, total, err := s.repo.GetAll(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch markets: %w", err)
	}

	// Add current pricing to markets
	for i := range markets {
		s.enrichMarketWithPricing(&markets[i])
	}

	return &MarketListResponse{
		Markets: ToMarketResponseList(markets),
		Total:   total,
		Page:    filters.Page,
		PerPage: filters.PerPage,
	}, nil
}

// GetMarketByID returns detailed market information
func (s *service) GetMarketByID(ctx context.Context, id uuid.UUID) (*MarketDetailResponse, error) {
	market, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to fetch market: %w", err)
	}

	s.enrichMarketWithPricing(market)

	go s.incrementViewCount(context.Background(), id)

	return ToMarketDetailResponse(market), nil
}

// GetMarketsByCategory returns markets for a specific category
func (s *service) GetMarketsByCategory(ctx context.Context, categoryID uuid.UUID) ([]MarketResponse, error) {
	filters := MarketFilters{
		CategoryID: &categoryID,
		Status:     &[]models.MarketStatus{models.MarketStatusOpen}[0],
	}

	markets, _, err := s.repo.GetAll(ctx, &filters)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch markets by category: %w", err)
	}

	return ToMarketResponseList(markets), nil
}

// GetMyMarkets returns markets created by a specific user
func (s *service) GetMyMarkets(ctx context.Context, userID uuid.UUID) ([]MarketResponse, error) {
	markets, err := s.repo.GetByCreator(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user markets: %w", err)
	}

	return ToMarketResponseList(markets), nil
}

// CreateMarket creates a new prediction market
func (s *service) CreateMarket(ctx context.Context, req *CreateMarketRequest) (*MarketDetailResponse, error) {
	if err := s.validateMarketTiming(req.CloseTime, req.ResolutionDeadline); err != nil {
		return nil, err
	}

	if err := s.validateOutcomes(req.Outcomes); err != nil {
		return nil, err
	}

	market := &models.Market{
		CountryID:           req.CountryID,
		CategoryID:          req.CategoryID,
		Title:               strings.TrimSpace(req.Title),
		Description:         strings.TrimSpace(req.Description),
		MarketType:          models.MarketType(req.MarketType),
		Status:              models.MarketStatusDraft,
		CloseTime:           req.CloseTime,
		ResolutionDeadline:  req.ResolutionDeadline,
		MinBetAmount:        s.getMinBetAmount(req.MinBetAmount),
		MaxBetAmount:        req.MaxBetAmount,
		RakePercentage:      s.getRakePercentage(req.RakePercentage),
		CreatorRevenueShare: s.getCreatorRevenueShare(req.CreatorRevenueShare),
		SafeguardConfig:     s.buildSafeguardConfig(req.SafeguardConfig),
		OracleConfig:        s.buildOracleConfig(req.OracleConfig),
		Metadata:            s.buildMarketMetadata(req.Tags),
	}

	// Set creator if available from context
	if creatorID := s.getCreatorFromContext(ctx); creatorID != uuid.Nil {
		market.CreatorID = &creatorID
	}

	// Validate market
	if err := market.Validate(); err != nil {
		return nil, err
	}

	// Create market in transaction
	err := s.repo.Create(ctx, market)
	if err != nil {
		return nil, fmt.Errorf("failed to create market: %w", err)
	}

	// Create outcomes
	for i, outcomeReq := range req.Outcomes {
		outcome := &models.MarketOutcome{
			MarketID:     market.ID,
			OutcomeKey:   strings.ToLower(strings.TrimSpace(outcomeReq.OutcomeKey)),
			OutcomeLabel: strings.TrimSpace(outcomeReq.OutcomeLabel),
			SortOrder:    outcomeReq.SortOrder,
		}

		if outcome.SortOrder == 0 {
			outcome.SortOrder = i + 1
		}

		if err := s.repo.CreateMarketOutcome(ctx, outcome); err != nil {
			return nil, fmt.Errorf("failed to create market outcome: %w", err)
		}

		market.Outcomes = append(market.Outcomes, *outcome)
	}

	// Set market status to open if moderation is not required
	if !s.config.RequireModeration {
		market.Status = models.MarketStatusOpen
		if err := s.repo.Update(ctx, market); err != nil {
			return nil, fmt.Errorf("failed to update market status: %w", err)
		}
	}

	return ToMarketDetailResponse(market), nil
}

// UpdateMarket updates an existing market
func (s *service) UpdateMarket(ctx context.Context, id uuid.UUID, req *UpdateMarketRequest) (*MarketDetailResponse, error) {
	market, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to fetch market: %w", err)
	}

	// Check if market can be updated
	if !s.canUpdateMarket(market) {
		return nil, errors.New("market cannot be updated in current status")
	}

	// Update fields if provided
	s.updateMarketFields(market, req)

	// Validate updated market
	if err := market.Validate(); err != nil {
		return nil, err
	}

	// Save updates
	if err := s.repo.Update(ctx, market); err != nil {
		return nil, fmt.Errorf("failed to update market: %w", err)
	}

	return ToMarketDetailResponse(market), nil
}

// ResolveMarket resolves a prediction market
func (s *service) ResolveMarket(ctx context.Context, id uuid.UUID, req ResolveMarketRequest) (*MarketDetailResponse, error) {
	market, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to fetch market: %w", err)
	}

	// Check if market can be resolved
	if !market.CanResolve() {
		return nil, errors.New("market cannot be resolved in current status")
	}

	// Validate winning outcome exists
	var winningOutcome *models.MarketOutcome
	for i := range market.Outcomes {
		if market.Outcomes[i].OutcomeKey == req.WinningOutcome {
			winningOutcome = &market.Outcomes[i]
			break
		}
	}

	if winningOutcome == nil {
		return nil, errors.New("invalid winning outcome")
	}

	// Resolve the market
	if err := market.Resolve(req.WinningOutcome, req.ResolutionSource); err != nil {
		return nil, fmt.Errorf("failed to resolve market: %w", err)
	}

	// Mark winning outcome
	winningOutcome.SetAsWinner()
	if err := s.repo.UpdateMarketOutcome(ctx, winningOutcome); err != nil {
		return nil, fmt.Errorf("failed to update winning outcome: %w", err)
	}

	// Mark losing outcomes
	for i := range market.Outcomes {
		if market.Outcomes[i].OutcomeKey != req.WinningOutcome {
			market.Outcomes[i].SetAsLoser()
			if err := s.repo.UpdateMarketOutcome(ctx, &market.Outcomes[i]); err != nil {
				return nil, fmt.Errorf("failed to update losing outcome: %w", err)
			}
		}
	}

	// Save market
	if err := s.repo.Update(ctx, market); err != nil {
		return nil, fmt.Errorf("failed to save resolved market: %w", err)
	}

	// TODO: Trigger settlement process (async)
	go s.processMarketSettlement(context.Background(), market.ID)

	return ToMarketDetailResponse(market), nil
}

// VoidMarket voids a market (refunds all bets)
func (s *service) VoidMarket(ctx context.Context, id uuid.UUID, reason string) error {
	market, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ErrRecordNotFound
		}
		return fmt.Errorf("failed to fetch market: %w", err)
	}

	// Void the market
	if err := market.Void(); err != nil {
		return err
	}

	// Update market with void reason
	market.ResolutionSource = fmt.Sprintf("VOIDED: %s", reason)

	if err := s.repo.Update(ctx, market); err != nil {
		return fmt.Errorf("failed to void market: %w", err)
	}

	// TODO: Trigger refund process (async)
	go s.processMarketRefunds(context.Background(), market.ID)

	return nil
}

// DeleteMarket deletes a market (only if no bets)
func (s *service) DeleteMarket(ctx context.Context, id uuid.UUID) error {
	market, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ErrRecordNotFound
		}
		return fmt.Errorf("failed to fetch market: %w", err)
	}

	// Check if market can be deleted
	if market.TotalPoolAmount.GreaterThan(decimal.Zero) {
		return errors.New("cannot delete market with existing bets")
	}

	if market.Status != models.MarketStatusDraft {
		return errors.New("can only delete draft markets")
	}

	return s.repo.Delete(ctx, id)
}

// AddMarketOutcome adds a new outcome to a market
func (s *service) AddMarketOutcome(ctx context.Context, marketID uuid.UUID, req CreateOutcomeRequest) (*OutcomeResponse, error) {
	market, err := s.repo.GetByID(ctx, marketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to fetch market: %w", err)
	}

	// Check if outcome can be added
	if market.Status != models.MarketStatusDraft {
		return nil, errors.New("can only add outcomes to draft markets")
	}

	// Check for duplicate outcome key
	for i := range market.Outcomes {
		outcome := &market.Outcomes[i]
		if strings.EqualFold(outcome.OutcomeKey, req.OutcomeKey) {
			return nil, errors.New("outcome key already exists")
		}
	}

	outcome := &models.MarketOutcome{
		MarketID:     marketID,
		OutcomeKey:   strings.ToLower(strings.TrimSpace(req.OutcomeKey)),
		OutcomeLabel: strings.TrimSpace(req.OutcomeLabel),
		SortOrder:    req.SortOrder,
	}

	if err := s.repo.CreateMarketOutcome(ctx, outcome); err != nil {
		return nil, fmt.Errorf("failed to create outcome: %w", err)
	}

	return ToOutcomeResponse(outcome), nil
}

// UpdateMarketOutcome updates an existing market outcome
func (s *service) UpdateMarketOutcome(ctx context.Context, outcomeID uuid.UUID, _ UpdateOutcomeRequest) (*OutcomeResponse, error) {
	// First get the outcome to find the market
	_, err := s.repo.GetMarketOutcomes(ctx, outcomeID) // This needs to be fixed - we need GetOutcomeByID
	if err != nil {
		return nil, fmt.Errorf("failed to fetch outcome: %w", err)
	}

	// For now, return error - we need to implement GetOutcomeByID in repository
	return nil, errors.New("update outcome not implemented - need GetOutcomeByID in repository")
}

// DeleteMarketOutcome deletes a market outcome
func (s *service) DeleteMarketOutcome(_ context.Context, _ uuid.UUID) error {
	// Similar issue - need GetOutcomeByID to check if deletion is allowed
	return errors.New("delete outcome not implemented - need GetOutcomeByID in repository")
}

// CalculateCurrentPrices calculates current prices for all market outcomes
func (s *service) CalculateCurrentPrices(ctx context.Context, marketID uuid.UUID) (map[string]PriceInfo, error) {
	market, err := s.repo.GetByID(ctx, marketID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market: %w", err)
	}

	prices := make(map[string]PriceInfo)
	totalPool := market.TotalPoolAmount.InexactFloat64()

	for i := range market.Outcomes {
		outcome := &market.Outcomes[i]

		outcomePool := outcome.PoolAmount.InexactFloat64()
		currentPrice := s.pricingEngine.CalculatePrice(totalPool, outcomePool)

		prices[outcome.OutcomeKey] = PriceInfo{
			CurrentPrice:   decimal.NewFromFloat(currentPrice),
			PriceChange24h: decimal.Zero, // TODO: Implement price history
			Volume24h:      decimal.Zero, // TODO: Implement volume tracking
			LastTradePrice: decimal.NewFromFloat(currentPrice),
		}
	}

	return prices, nil
}

// CheckSafeguards checks the current safeguard status of a market
func (s *service) CheckSafeguards(ctx context.Context, marketID uuid.UUID) (*SafeguardStatus, error) {
	market, err := s.repo.GetByID(ctx, marketID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market: %w", err)
	}

	status := &SafeguardStatus{
		QuorumMet:         s.safeguardEngine.CheckQuorum(market, market.Outcomes),
		IsBalanced:        s.safeguardEngine.CheckImbalance(market.Outcomes, market.SafeguardConfig.ImbalanceThreshold.InexactFloat64()),
		HouseBotActive:    s.safeguardEngine.ShouldTriggerHouseBot(market, market.Outcomes),
		VoidRisk:          false,
		CurrentQuorum:     market.TotalPoolAmount,
		ImbalanceRatio:    s.calculateImbalanceRatio(market.Outcomes),
		RecommendedAction: "none",
	}

	// Determine void risk
	if !status.QuorumMet && market.CloseTime.Before(time.Now().Add(24*time.Hour)) {
		status.VoidRisk = true
		status.RecommendedAction = "increase_promotion"
	}

	return status, nil
}

// ProcessExpiredMarkets processes markets that have expired
func (s *service) ProcessExpiredMarkets(ctx context.Context) error {
	expiredMarkets, err := s.repo.GetExpiredMarkets(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch expired markets: %w", err)
	}

	for i := range expiredMarkets {
		market := expiredMarkets[i]
		market.Status = models.MarketStatusClosed
		if err := s.repo.Update(ctx, &market); err != nil {
			continue
		}

		// Check if market should be auto-resolved
		if market.OracleConfig.AutoResolve {
			// TODO: Trigger oracle resolution
			go s.processOracleResolution(context.Background(), market.ID)
		}
	}

	return nil
}

func (s *service) validateMarketTiming(closeTime, resolutionDeadline time.Time) error {
	now := time.Now()

	if closeTime.Before(now.Add(s.config.MinMarketDuration)) {
		return fmt.Errorf("market close time must be at least %v from now", s.config.MinMarketDuration)
	}

	if closeTime.After(now.Add(s.config.MaxMarketDuration)) {
		return fmt.Errorf("market close time cannot be more than %v from now", s.config.MaxMarketDuration)
	}

	if resolutionDeadline.Before(closeTime.Add(time.Hour)) {
		return errors.New("resolution deadline must be at least 1 hour after close time")
	}

	return nil
}

func (s *service) validateOutcomes(outcomes []CreateOutcomeRequest) error {
	if len(outcomes) < 2 {
		return errors.New("market must have at least 2 outcomes")
	}

	if len(outcomes) > 10 {
		return errors.New("market cannot have more than 10 outcomes")
	}

	keys := make(map[string]bool)
	for _, outcome := range outcomes {
		key := strings.ToLower(strings.TrimSpace(outcome.OutcomeKey))
		if keys[key] {
			return errors.New("duplicate outcome keys not allowed")
		}
		keys[key] = true

		if outcome.OutcomeKey == "" || outcome.OutcomeLabel == "" {
			return errors.New("outcome key and label are required")
		}
	}

	return nil
}

func (s *service) getMinBetAmount(amount decimal.Decimal) decimal.Decimal {
	if amount.IsZero() {
		return s.config.MinBetAmount
	}
	return amount
}

func (s *service) getRakePercentage(percentage *decimal.Decimal) decimal.Decimal {
	if percentage == nil {
		return s.config.DefaultRakePercentage
	}
	return *percentage
}

func (s *service) getCreatorRevenueShare(share *decimal.Decimal) decimal.Decimal {
	if share == nil {
		return s.config.DefaultCreatorRevenueShare
	}
	return *share
}

func (s *service) buildSafeguardConfig(req *CreateSafeguardConfigRequest) models.SafeguardConfig {
	if req == nil {
		return models.SafeguardConfig{
			MinQuorumAmount:    s.config.MinQuorumAmount,
			MinOutcomes:        2,
			HouseBotEnabled:    s.config.EnableHouseBot,
			HouseBotAmount:     s.config.HouseBotAmount,
			ImbalanceThreshold: decimal.NewFromFloat(0.8),
			VoidOnQuorumFail:   true,
		}
	}

	return models.SafeguardConfig{
		MinQuorumAmount:    req.MinQuorumAmount,
		MinOutcomes:        req.MinOutcomes,
		HouseBotEnabled:    req.HouseBotEnabled,
		HouseBotAmount:     req.HouseBotAmount,
		ImbalanceThreshold: req.ImbalanceThreshold,
		VoidOnQuorumFail:   req.VoidOnQuorumFail,
	}
}

func (s *service) buildOracleConfig(req *CreateOracleConfigRequest) models.OracleConfig {
	if req == nil {
		return models.OracleConfig{}
	}

	return models.OracleConfig{
		Provider:       req.Provider,
		DataSource:     req.DataSource,
		ResolutionURL:  req.ResolutionURL,
		Criteria:       req.Criteria,
		AutoResolve:    req.AutoResolve,
		BackupProvider: req.BackupProvider,
	}
}

func (s *service) buildMarketMetadata(tags []string) models.MarketMetadata {
	return models.MarketMetadata{
		Tags:      tags,
		ViewCount: 0,
	}
}

func (s *service) getCreatorFromContext(_ context.Context) uuid.UUID {
	// TODO: Extract user ID from JWT context
	return uuid.Nil
}

func (s *service) canUpdateMarket(market *models.Market) bool {
	// Only draft and open markets can be updated
	return market.Status == models.MarketStatusDraft || market.Status == models.MarketStatusOpen
}

func (s *service) updateMarketFields(market *models.Market, req *UpdateMarketRequest) {
	if req.Title != nil {
		market.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		market.Description = strings.TrimSpace(*req.Description)
	}
	if req.CloseTime != nil {
		market.CloseTime = *req.CloseTime
	}
	if req.ResolutionDeadline != nil {
		market.ResolutionDeadline = *req.ResolutionDeadline
	}
	if req.MinBetAmount != nil {
		market.MinBetAmount = *req.MinBetAmount
	}
	if req.MaxBetAmount != nil {
		market.MaxBetAmount = req.MaxBetAmount
	}
	if req.SafeguardConfig != nil {
		market.SafeguardConfig = s.buildSafeguardConfig(req.SafeguardConfig)
	}
	if req.OracleConfig != nil {
		market.OracleConfig = s.buildOracleConfig(req.OracleConfig)
	}
	if req.Tags != nil {
		market.Metadata.Tags = req.Tags
	}
}

func (s *service) enrichMarketWithPricing(market *models.Market) {
	// Calculate current prices for outcomes
	for i := range market.Outcomes {
		price := market.Outcomes[i].GetCurrentPrice(market.TotalPoolAmount)
		market.Outcomes[i].PoolAmount = price
	}
}

func (s *service) calculateImbalanceRatio(outcomes []models.MarketOutcome) decimal.Decimal {
	if len(outcomes) == 0 {
		return decimal.Zero
	}

	var maxPool, totalPool decimal.Decimal

	for i := range outcomes {
		pool := outcomes[i].PoolAmount

		totalPool = totalPool.Add(pool)
		if pool.GreaterThan(maxPool) {
			maxPool = pool
		}
	}

	if totalPool.IsZero() {
		return decimal.Zero
	}
	return maxPool.Div(totalPool)
}

func (s *service) incrementViewCount(_ context.Context, _ uuid.UUID) {
	// TODO: Implement view count increment
}

func (s *service) processMarketSettlement(_ context.Context, _ uuid.UUID) {
	// TODO: Implement market settlement process
}

func (s *service) processMarketRefunds(_ context.Context, _ uuid.UUID) {
	// TODO: Implement market refund process
}

func (s *service) processOracleResolution(_ context.Context, _ uuid.UUID) {
	// TODO: Implement oracle resolution process
}
