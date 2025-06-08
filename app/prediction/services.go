package prediction

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/joefazee/neo/models"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// service implements the Service interface
type service struct {
	db            *gorm.DB // Main DB connection for starting transactions
	repo          Repository
	config        *Config
	bettingEngine BettingEngine
	riskEngine    RiskEngine
	validator     *validator.Validate
}

// NewService creates a new betting service
func NewService(db *gorm.DB, repo Repository, config *Config, bettingEngine BettingEngine, riskEngine RiskEngine) Service {
	return &service{
		db:            db,
		repo:          repo,
		config:        config,
		bettingEngine: bettingEngine,
		riskEngine:    riskEngine,
		validator:     validator.New(),
	}
}

// PlaceBet places a new bet after performing validations and risk checks.
// It ensures all database operations are performed within a single transaction.
func (s *service) PlaceBet(
	ctx context.Context,
	userID uuid.UUID,
	req *PlaceBetRequest,
) (*BetResponse, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	market, outcome, err := s.loadMarketAndOutcome(ctx, req.MarketID, req.OutcomeID)
	if err != nil {
		return nil, err
	}

	if market.Country == nil {
		log.Printf("Warning: Market %s has no associated country data for currency code.", market.ID)
		return nil, errors.New("market configuration error: missing country data for currency")
	}
	currency := market.Country.CurrencyCode

	if err := s.runRiskChecks(ctx, userID, market, req.Amount, currency); err != nil {
		return nil, err
	}

	price, contracts, err := s.determinePriceAndContracts(
		market, outcome, req.Amount,
		req.ExpectedPrice, req.MaxSlippage,
	)
	if err != nil {
		return nil, err
	}

	if contracts.IsZero() && req.Amount.GreaterThan(decimal.Zero) {
		return nil, errors.New("bet amount too small to purchase any contracts at current price, or price is too extreme")
	}

	bet, err := s.createBetTransaction(
		ctx, userID, market, outcome,
		req.Amount, contracts, price, currency,
	)
	if err != nil {
		// The error from createBetTransaction will already be descriptive.
		return nil, fmt.Errorf("failed to execute bet transaction: %w", err)
	}

	resp := ToBetResponse(bet)
	// Populate dynamic fields based on the price at the time of the bet
	if bet.Market != nil && bet.MarketOutcome != nil {
		resp.CurrentPrice = price // This is the price at which the bet was executed
		resp.PotentialPayout = s.calculatePotentialPayout(bet)
		resp.ProfitLoss = s.calculateCurrentProfitLoss(bet, price) // P&L if current price was execution price
	}

	return resp, nil
}

// loadMarketAndOutcome fetches the market (with outcomes and country) and the specific outcome.
func (s *service) loadMarketAndOutcome(
	ctx context.Context, marketID, outcomeID uuid.UUID,
) (*models.Market, *models.MarketOutcome, error) {
	market, err := s.repo.GetMarketWithOutcomes(ctx, marketID) // Expects Country to be preloaded
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, models.ErrRecordNotFound
		}
		return nil, nil, fmt.Errorf("get market %s: %w", marketID, err)
	}

	var foundOutcome *models.MarketOutcome
	for i := range market.Outcomes {
		if market.Outcomes[i].ID == outcomeID {
			foundOutcome = &market.Outcomes[i]
			break
		}
	}

	if foundOutcome == nil {
		return nil, nil, fmt.Errorf("outcome %s not found within market %s: %w", outcomeID, marketID, models.ErrRecordNotFound)
	}
	// The MarketID check is implicitly handled by how foundOutcome is derived from market.Outcomes.

	return market, foundOutcome, nil
}

// runRiskChecks performs all necessary risk evaluations before placing a bet.
func (s *service) runRiskChecks(
	ctx context.Context,
	userID uuid.UUID,
	market *models.Market,
	amount decimal.Decimal,
	currencyCode string,
) error {
	user := s.getUserByID(ctx, userID) // This is a placeholder in the service

	checks := []func() error{
		func() error { return s.riskEngine.ValidateMarketForBetting(market) },
		func() error { return s.riskEngine.ValidateUserForBetting(user) },
		func() error { return s.riskEngine.CheckBettingLimits(userID, amount, market) },
		func() error { return s.riskEngine.CheckPositionLimits(userID, amount, market) },
		func() error { return s.riskEngine.CheckRateLimit(userID) },
		func() error { return s.riskEngine.CheckCooldown(userID) },
		func() error { return s.riskEngine.CheckWalletBalance(userID, amount, currencyCode) },
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

// determinePriceAndContracts calculates the execution price and contracts, handling slippage.
func (s *service) determinePriceAndContracts(
	market *models.Market,
	outcome *models.MarketOutcome,
	amount, expectedPrice, maxSlippage decimal.Decimal,
) (price, contracts decimal.Decimal, err error) {
	price = s.bettingEngine.CalculateContractPrice(market, outcome)

	if s.config.EnableSlippageProtection && !expectedPrice.IsZero() {
		slippage := s.bettingEngine.CalculateSlippage(expectedPrice, price)
		effectiveMaxSlippage := maxSlippage
		if effectiveMaxSlippage.IsZero() {
			effectiveMaxSlippage = s.config.MaxSlippagePercentage
		}
		if err = s.bettingEngine.ValidateSlippage(slippage, effectiveMaxSlippage); err != nil {
			return decimal.Zero, decimal.Zero, fmt.Errorf("slippage validation failed: %w", err)
		}
	}

	contracts = s.bettingEngine.CalculateContractsBought(amount, price)
	return price, contracts, nil
}

// createBetTransaction handles the database operations for creating a bet atomically.
func (s *service) createBetTransaction(ctx context.Context,
	userID uuid.UUID,
	market *models.Market,
	outcome *models.MarketOutcome,
	amount, contracts, price decimal.Decimal,
	currencyCode string) (*models.Bet, error) {
	var betRecordToReturn *models.Bet

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repoTx := s.repo.WithTx(tx)

		wallet, err := repoTx.GetUserWallet(ctx, userID, currencyCode)
		if err != nil {
			return fmt.Errorf("get user wallet: %w", err)
		}

		if !wallet.CanDebit(amount) {
			return models.ErrInsufficientWalletBalance
		}
		originalBalance := wallet.Balance

		ledgerTx := models.CreateBetTransaction(userID, wallet.ID, amount, originalBalance, uuid.Nil)
		if err := repoTx.CreateTransaction(ctx, ledgerTx); err != nil {
			return fmt.Errorf("create ledger transaction: %w", err)
		}

		bet := &models.Bet{
			UserID:           userID,
			MarketID:         market.ID,
			MarketOutcomeID:  outcome.ID,
			Amount:           amount,
			ContractsBought:  contracts,
			PricePerContract: price,
			TotalCost:        amount,
			TransactionID:    ledgerTx.ID,
			Status:           models.BetStatusActive,
		}
		if err := repoTx.CreateBet(ctx, bet); err != nil {
			return fmt.Errorf("create bet record: %w", err)
		}
		betRecordToReturn = bet

		ledgerTx.ReferenceID = &bet.ID
		if err := repoTx.UpdateTransaction(ctx, ledgerTx); err != nil {
			return fmt.Errorf("update ledger transaction with bet ID: %w", err)
		}

		if err := wallet.Debit(amount); err != nil {
			return fmt.Errorf("in-memory wallet debit: %w", err)
		}
		if err := repoTx.UpdateWallet(ctx, wallet); err != nil {
			return fmt.Errorf("update wallet record: %w", err)
		}

		// 5. Update Market and Outcome Pools
		// These should ideally be fetched fresh within the transaction or locked.
		// For MVP, updating passed-in objects assuming they are current for this tx.
		market.TotalPoolAmount = market.TotalPoolAmount.Add(amount)
		outcome.PoolAmount = outcome.PoolAmount.Add(amount)

		if err := repoTx.UpdateMarket(ctx, market); err != nil {
			return fmt.Errorf("update market pool: %w", err)
		}
		if err := repoTx.UpdateMarketOutcome(ctx, outcome); err != nil {
			return fmt.Errorf("update outcome pool: %w", err)
		}

		// Populate associations for the response
		betRecordToReturn.Market = market
		betRecordToReturn.MarketOutcome = outcome
		betRecordToReturn.Transaction = ledgerTx

		return nil // Commit transaction
	})

	if err != nil {
		return nil, err
	}
	return betRecordToReturn, nil
}

// CancelBet cancels an active bet if within the allowed window and refunds the user.
// This operation is transactional.
func (s *service) CancelBet(ctx context.Context, userID, betID uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repoTx := s.repo.WithTx(tx)

		// 1) fetch + validate all preconditions
		bet, currency, err := s.fetchAndValidateCancel(ctx, repoTx, userID, betID)
		if err != nil {
			return err
		}

		// 2) perform the refund (all DB updates + wallet ops)
		if err := s.executeRefund(ctx, repoTx, bet, currency); err != nil {
			return err
		}

		log.Printf("Bet %s for user %s canceled and refunded %s",
			bet.ID, bet.UserID, bet.Amount)
		return nil
	})
}

// fetchAndValidateCancel loads the bet and runs every single pre-refund check.
// On error it returns the right models.Err… or fmt.Errorf wrap.
func (s *service) fetchAndValidateCancel(
	ctx context.Context,
	repoTx Repository,
	userID, betID uuid.UUID,
) (*models.Bet, string, error) {
	bet, err := repoTx.GetBetByID(ctx, betID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", models.ErrRecordNotFound
		}
		return nil, "", fmt.Errorf("get bet for cancellation: %w", err)
	}
	if bet.UserID != userID {
		return nil, "", models.ErrForbidden
	}
	if bet.Status != models.BetStatusActive {
		return nil, "", errors.New("bet is not active and cannot be canceled")
	}

	window := s.config.BetCancellationWindow
	if window <= 0 {
		window = 5 * time.Minute
	}
	if time.Since(bet.CreatedAt) > window {
		return nil, "", fmt.Errorf(
			"bet cancellation period of %v has expired", window,
		)
	}

	if bet.Market == nil || bet.Market.Country == nil {
		return nil, "", errors.New(
			"cannot process refund: missing market currency information",
		)
	}

	return bet, bet.Market.Country.CurrencyCode, nil
}

// executeRefund does all of the DB + wallet + pool updates for the refund.
// It returns the first error it encounters, exactly as before.
func (s *service) executeRefund(
	ctx context.Context,
	repoTx Repository,
	bet *models.Bet,
	currencyCode string,
) error {
	// 1) mark refunded
	bet.Status = models.BetStatusRefunded
	now := time.Now()
	bet.SettledAt = &now
	amount := bet.Amount
	bet.SettlementAmount = &amount

	if err := repoTx.UpdateBet(ctx, bet); err != nil {
		return fmt.Errorf("update bet status to refunded: %w", err)
	}

	// 2) ledger tx
	wallet, err := repoTx.GetUserWallet(ctx, bet.UserID, currencyCode)
	if err != nil {
		return fmt.Errorf("get user wallet for refund: %w", err)
	}
	original := wallet.Balance

	refundTx := models.CreateBetRefundTransaction(
		bet.UserID, wallet.ID, amount, original, bet.ID,
	)
	if err := repoTx.CreateTransaction(ctx, refundTx); err != nil {
		return fmt.Errorf("create refund ledger transaction: %w", err)
	}

	// 3) credit + persist wallet
	if err := wallet.Credit(amount); err != nil {
		return fmt.Errorf("in-memory wallet credit for refund: %w", err)
	}
	if err := repoTx.UpdateWallet(ctx, wallet); err != nil {
		return fmt.Errorf("update wallet record for refund: %w", err)
	}

	// 4) adjust pools
	if bet.MarketOutcome != nil {
		m := bet.Market
		o := bet.MarketOutcome
		m.TotalPoolAmount = m.TotalPoolAmount.Sub(amount)
		o.PoolAmount = o.PoolAmount.Sub(amount)

		if err := repoTx.UpdateMarket(ctx, m); err != nil {
			return fmt.Errorf("update market pool on refund: %w", err)
		}
		if err := repoTx.UpdateMarketOutcome(ctx, o); err != nil {
			return fmt.Errorf("update outcome pool on refund: %w", err)
		}
	} else {
		log.Printf(
			"Warning: MarketOutcome data missing for bet %s during refund; pools not adjusted.",
			bet.ID,
		)
	}

	return nil
}

// GetBetByID returns a specific bet, ensuring ownership.
func (s *service) GetBetByID(ctx context.Context, userID, betID uuid.UUID) (*BetResponse, error) {
	bet, err := s.repo.GetBetByID(ctx, betID) // Preloads Market.Country
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get bet: %w", err)
	}

	if bet.UserID != userID {
		return nil, models.ErrForbidden
	}

	response := ToBetResponse(bet)

	if bet.Status == models.BetStatusActive && bet.Market != nil && bet.MarketOutcome != nil {
		if bet.Market.Country == nil { // Should be preloaded, but defensive check
			log.Printf("Warning: Missing Country for Market %s in GetBetByID. Price calculations might be affected.", bet.Market.ID)
		}
		currentPrice := s.bettingEngine.CalculateContractPrice(bet.Market, bet.MarketOutcome)
		response.CurrentPrice = currentPrice
		response.PotentialPayout = s.calculatePotentialPayout(bet)
		response.ProfitLoss = s.calculateCurrentProfitLoss(bet, currentPrice)
	}
	return response, nil
}

// GetUserBets returns paginated user bets with current price info for active bets.
func (s *service) GetUserBets(ctx context.Context, userID uuid.UUID, filters *BetFilters) (*BetListResponse, error) {
	if filters.Page <= 0 {
		filters.Page = 1
	}
	if filters.PerPage <= 0 || filters.PerPage > 100 {
		filters.PerPage = 20
	}

	bets, total, err := s.repo.GetBetsByUser(ctx, userID, filters) // Preloads Market.Country
	if err != nil {
		return nil, fmt.Errorf("failed to get user bets: %w", err)
	}

	responses := make([]BetResponse, len(bets))
	for i := range bets {
		bet := bets[i]
		response := ToBetResponse(&bet)

		if bet.Status == models.BetStatusActive && bet.Market != nil && bet.MarketOutcome != nil {
			if bet.Market.Country == nil {
				log.Printf("Warning: Missing Country for Market %s in GetUserBets. Calculations might be affected.", bet.Market.ID)
			}
			currentPrice := s.bettingEngine.CalculateContractPrice(bet.Market, bet.MarketOutcome)
			response.CurrentPrice = currentPrice
			response.PotentialPayout = s.calculatePotentialPayout(&bet)
			response.ProfitLoss = s.calculateCurrentProfitLoss(&bet, currentPrice)
		}
		responses[i] = *response
	}

	return &BetListResponse{
		Bets:    responses,
		Total:   total,
		Page:    filters.Page,
		PerPage: filters.PerPage,
	}, nil
}

// GetUserPositions returns user's current positions across active markets.
func (s *service) GetUserPositions(ctx context.Context, userID uuid.UUID) ([]PositionResponse, error) {
	activeBets, err := s.repo.GetActiveBetsByUser(ctx, userID) // Preloads Market.Country
	if err != nil {
		return nil, fmt.Errorf("failed to get active bets: %w", err)
	}

	positionsMap := make(map[string]*PositionResponse)

	for i := range activeBets {
		bet := &activeBets[i]
		if bet.Market == nil || bet.MarketOutcome == nil || bet.Market.Country == nil {
			log.Printf("Warning: Bet %s is missing Market, MarketOutcome, or Market.Country data, skipping for position calculation.", bet.ID)
			continue
		}

		key := fmt.Sprintf("%s-%s", bet.MarketID, bet.MarketOutcomeID)

		if position, exists := positionsMap[key]; exists {
			position.TotalInvested = position.TotalInvested.Add(bet.Amount)
			position.TotalContracts = position.TotalContracts.Add(bet.ContractsBought)
			position.BetCount++
			if bet.CreatedAt.After(position.LastBetAt) {
				position.LastBetAt = bet.CreatedAt
			}
		} else {
			positionsMap[key] = &PositionResponse{
				MarketID:       bet.MarketID,
				MarketTitle:    bet.Market.Title,
				OutcomeID:      bet.MarketOutcomeID,
				OutcomeLabel:   bet.MarketOutcome.OutcomeLabel,
				TotalInvested:  bet.Amount,
				TotalContracts: bet.ContractsBought,
				BetCount:       1,
				LastBetAt:      bet.CreatedAt,
			}
		}
	}

	result := make([]PositionResponse, 0, len(positionsMap))
	for _, position := range positionsMap {
		if position.TotalContracts.GreaterThan(decimal.Zero) {
			position.AveragePrice = position.TotalInvested.Div(position.TotalContracts).Mul(decimal.NewFromInt(100))
		}

		outcome, errOutcome := s.repo.GetMarketOutcome(ctx, position.OutcomeID) // Preloads Market.Country
		if errOutcome == nil && outcome != nil && outcome.Market != nil && outcome.Market.Country != nil {
			position.CurrentPrice = s.bettingEngine.CalculateContractPrice(outcome.Market, outcome)
			position.CurrentValue = position.TotalContracts.Mul(position.CurrentPrice.Div(decimal.NewFromInt(100)))
			position.ProfitLoss = position.CurrentValue.Sub(position.TotalInvested)

			if position.TotalInvested.GreaterThan(decimal.Zero) {
				position.ProfitLossPercent = position.ProfitLoss.Div(position.TotalInvested).Mul(decimal.NewFromInt(100))
			}
		} else if errOutcome != nil {
			log.Printf("Warning: Failed to get outcome %s for position calculation: %v", position.OutcomeID, errOutcome)
		}
		result = append(result, *position)
	}
	return result, nil
}

// CalculateBetQuote calculates a quote for a potential bet.
func (s *service) CalculateBetQuote(ctx context.Context, req BetQuoteRequest) (*BetQuoteResponse, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	market, err := s.repo.GetMarketWithOutcomes(ctx, req.MarketID) // Preloads Country
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get market: %w", err)
	}
	if market.Country == nil {
		return nil, errors.New("market configuration error: missing country data for quote")
	}

	var outcome *models.MarketOutcome
	for i := range market.Outcomes {
		if market.Outcomes[i].ID == req.OutcomeID {
			outcome = &market.Outcomes[i]
			break
		}
	}
	if outcome == nil {
		return nil, fmt.Errorf("outcome %s not found in market %s: %w", req.OutcomeID, req.MarketID, models.ErrRecordNotFound)
	}

	currentPrice := s.bettingEngine.CalculateContractPrice(market, outcome)
	estimatedPrice := s.bettingEngine.CalculateNewPrice(market, outcome, req.Amount)
	priceImpact := s.bettingEngine.CalculatePriceImpact(market.TotalPoolAmount, req.Amount)
	contractsBought := s.bettingEngine.CalculateContractsBought(req.Amount, currentPrice)

	var potentialPayout decimal.Decimal
	if !estimatedPrice.IsZero() {
		potentialPayout = contractsBought.Mul(decimal.NewFromInt(100)).Div(estimatedPrice)
	}

	breakevenPrice := s.bettingEngine.CalculateBreakevenPrice(req.Amount, contractsBought)
	slippage := s.bettingEngine.CalculateSlippage(currentPrice, estimatedPrice)

	var warnings []string
	if priceImpact.GreaterThan(s.config.SignificantPriceImpactThreshold) {
		warnings = append(warnings, "High price impact expected")
	}
	if slippage.GreaterThan(s.config.MaxSlippagePercentage) {
		warnings = append(warnings, "Slippage exceeds recommended limit")
	}

	return &BetQuoteResponse{
		MarketID:          req.MarketID,
		OutcomeID:         req.OutcomeID,
		Amount:            req.Amount,
		CurrentPrice:      currentPrice,
		EstimatedPrice:    estimatedPrice,
		PriceImpact:       priceImpact,
		ContractsBought:   contractsBought,
		PotentialPayout:   potentialPayout,
		BreakevenPrice:    breakevenPrice,
		MaxLoss:           req.Amount,
		EstimatedSlippage: slippage,
		ValidUntil:        time.Now().Add(time.Duration(s.config.BetTimeoutSeconds) * time.Second),
		Warnings:          warnings,
	}, nil
}

// GetMarketPriceImpact analyzes price impact for a potential bet.
func (s *service) GetMarketPriceImpact(ctx context.Context, marketID, outcomeID uuid.UUID, amount decimal.Decimal) (*PriceImpactResponse, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("amount must be positive for price impact calculation")
	}
	market, err := s.repo.GetMarketWithOutcomes(ctx, marketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get market: %w", err)
	}

	var outcome *models.MarketOutcome
	for i := range market.Outcomes {
		if market.Outcomes[i].ID == outcomeID {
			outcome = &market.Outcomes[i]
			break
		}
	}
	if outcome == nil {
		return nil, fmt.Errorf("outcome %s not found in market %s: %w", outcomeID, marketID, models.ErrRecordNotFound)
	}

	currentPrice := s.bettingEngine.CalculateContractPrice(market, outcome)
	newPrice := s.bettingEngine.CalculateNewPrice(market, outcome, amount)
	priceImpact := s.bettingEngine.CalculatePriceImpact(market.TotalPoolAmount, amount)

	impactCategory := "low"
	if priceImpact.GreaterThan(s.config.HighPriceImpactThreshold) {
		impactCategory = "high"
	} else if priceImpact.GreaterThan(s.config.ModeratePriceImpactThreshold) {
		impactCategory = "moderate"
	}

	return &PriceImpactResponse{
		MarketID:       marketID,
		OutcomeID:      outcomeID,
		BetAmount:      amount,
		CurrentPrice:   currentPrice,
		NewPrice:       newPrice,
		PriceImpact:    priceImpact,
		LiquidityDepth: market.TotalPoolAmount,
		ImpactCategory: impactCategory,
	}, nil
}

// GetUserPortfolio returns user's complete betting portfolio.
func (s *service) GetUserPortfolio(ctx context.Context, userID uuid.UUID) (*PortfolioResponse, error) {
	positions, err := s.GetUserPositions(ctx, userID)
	if err != nil {
		return nil, err
	}

	totalInvested := decimal.Zero
	currentValue := decimal.Zero
	marketsMap := make(map[uuid.UUID]bool)
	var lastActivity time.Time

	for i := range positions {
		position := &positions[i]
		totalInvested = totalInvested.Add(position.TotalInvested)
		currentValue = currentValue.Add(position.CurrentValue)
		marketsMap[position.MarketID] = true
		if position.LastBetAt.After(lastActivity) {
			lastActivity = position.LastBetAt
		}
	}

	totalProfitLoss := currentValue.Sub(totalInvested)
	profitLossPercent := decimal.Zero
	if totalInvested.GreaterThan(decimal.Zero) {
		profitLossPercent = totalProfitLoss.Div(totalInvested).Mul(decimal.NewFromInt(100))
	}

	winRate, err := s.calculateWinRate(ctx, userID)
	if err != nil {
		log.Printf("Warning: Failed to calculate win rate for user %s: %v", userID, err)
	}

	return &PortfolioResponse{
		UserID:            userID,
		TotalInvested:     totalInvested,
		CurrentValue:      currentValue,
		TotalProfitLoss:   totalProfitLoss,
		ProfitLossPercent: profitLossPercent,
		ActivePositions:   positions,
		TotalPositions:    len(positions),
		MarketsCount:      len(marketsMap),
		WinRate:           winRate,
		LastActivityAt:    lastActivity,
	}, nil
}

// GetUserBettingStats returns detailed betting statistics.
// --- in service.go ---

func (s *service) GetUserBettingStats(
	ctx context.Context,
	userID uuid.UUID,
) (*BettingStatsResponse, error) {
	filters := &BetFilters{Page: 1, PerPage: s.config.MaxBetsForStatsCalculation}
	bets, totalCount, err := s.repo.GetBetsByUser(ctx, userID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get user bets for stats: %w", err)
	}

	stats := &BettingStatsResponse{
		UserID:    userID,
		TotalBets: int(totalCount),
	}
	if len(bets) == 0 {
		return stats, nil
	}

	agg := aggregateBetStats(bets)

	stats.FirstBetAt = agg.FirstBetAt
	stats.LastBetAt = agg.LastBetAt
	stats.TotalAmount = agg.TotalAmount
	stats.WonBets = agg.Won
	stats.LostBets = agg.Lost
	stats.PendingBets = agg.Pending
	stats.TotalWinnings = agg.Winnings
	stats.TotalLosses = agg.Losses
	stats.NetProfit = agg.Winnings.Sub(agg.Losses)
	stats.LargestWin = agg.MaxWin
	stats.LargestLoss = agg.MaxLoss

	if settled := agg.Won + agg.Lost; settled > 0 {
		stats.WinRate = decimal.NewFromInt(int64(agg.Won)).
			Div(decimal.NewFromInt(int64(settled))).
			Mul(decimal.NewFromInt(100))
	}
	stats.AverageBetSize = agg.TotalAmount.Div(decimal.NewFromInt(totalCount))
	stats.ROI = stats.NetProfit.
		Div(agg.TotalAmount).
		Mul(decimal.NewFromInt(100))

	return stats, nil
}

type betAgg struct {
	FirstBetAt time.Time
	LastBetAt  time.Time

	TotalAmount        decimal.Decimal
	Won, Lost, Pending int

	Winnings, Losses decimal.Decimal
	MaxWin, MaxLoss  decimal.Decimal
}

func aggregateBetStats(bets []models.Bet) *betAgg {
	agg := &betAgg{
		TotalAmount: decimal.Zero,
		Winnings:    decimal.Zero,
		Losses:      decimal.Zero,
		MaxWin:      decimal.Zero,
		MaxLoss:     decimal.Zero,
	}

	for i := range bets {
		b := bets[i]
		agg.TotalAmount = agg.TotalAmount.Add(b.Amount)

		if agg.FirstBetAt.IsZero() || b.CreatedAt.Before(agg.FirstBetAt) {
			agg.FirstBetAt = b.CreatedAt
		}
		if b.CreatedAt.After(agg.LastBetAt) {
			agg.LastBetAt = b.CreatedAt
		}

		switch b.Status {
		case models.BetStatusSettled:
			if b.SettlementAmount == nil {
				continue
			}
			pnl := b.SettlementAmount.Sub(b.Amount)
			if pnl.GreaterThan(decimal.Zero) {
				agg.Won++
				agg.Winnings = agg.Winnings.Add(pnl)
				if pnl.GreaterThan(agg.MaxWin) {
					agg.MaxWin = pnl
				}
			} else if pnl.LessThan(decimal.Zero) {
				loss := pnl.Abs()
				agg.Lost++
				agg.Losses = agg.Losses.Add(loss)
				if loss.GreaterThan(agg.MaxLoss) {
					agg.MaxLoss = loss
				}
			}
		case models.BetStatusActive:
			agg.Pending++
		}
	}

	return agg
}

// getUserByID is a placeholder. In a real app, this would fetch from a user repository.
func (s *service) getUserByID(_ context.Context, userID uuid.UUID) *models.User {
	isActive := true
	return &models.User{
		ID:              userID,
		IsActive:        &isActive,
		EmailVerifiedAt: &[]time.Time{time.Now()}[0],
		KYCStatus:       models.KYCStatusVerified,
		KYCVerifiedAt:   &[]time.Time{time.Now()}[0],
		Country:         &models.Country{CurrencyCode: "USD"},
	}
}

func (s *service) calculatePotentialPayout(bet *models.Bet) decimal.Decimal {
	if bet.ContractsBought.IsZero() {
		return decimal.Zero
	}
	return bet.ContractsBought.Mul(decimal.NewFromInt(100))
}

func (s *service) calculateCurrentProfitLoss(bet *models.Bet, currentMarketPrice decimal.Decimal) decimal.Decimal {
	if bet.ContractsBought.IsZero() {
		if currentMarketPrice.IsZero() {
			return decimal.Zero
		}
		return bet.Amount.Neg()
	}
	if currentMarketPrice.IsZero() {
		return bet.Amount.Neg()
	}
	currentValue := bet.ContractsBought.Mul(currentMarketPrice.Div(decimal.NewFromInt(100)))
	return currentValue.Sub(bet.Amount)
}

func (s *service) calculateWinRate(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	filters := &BetFilters{Status: &[]models.BetStatus{models.BetStatusSettled}[0], Page: 1, PerPage: s.config.MaxBetsForStatsCalculation}
	bets, _, err := s.repo.GetBetsByUser(ctx, userID, filters)
	if err != nil {
		return decimal.Zero, fmt.Errorf("win rate - get bets: %w", err)
	}

	if len(bets) == 0 {
		return decimal.Zero, nil
	}

	wins := 0
	totalSettledForWinRate := 0
	for i := range bets {
		bet := &bets[i]
		if bet.Status == models.BetStatusSettled && bet.SettlementAmount != nil {
			totalSettledForWinRate++
			if bet.SettlementAmount.GreaterThan(bet.Amount) {
				wins++
			}
		}
	}

	if totalSettledForWinRate == 0 {
		return decimal.Zero, nil
	}

	return decimal.NewFromInt(int64(wins)).Div(decimal.NewFromInt(int64(totalSettledForWinRate))).Mul(decimal.NewFromInt(100)), nil
}
