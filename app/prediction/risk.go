package prediction

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/joefazee/neo/models"
	"github.com/shopspring/decimal"
)

// riskEngine implements the RiskEngine interface
type riskEngine struct {
	config *Config
	repo   Repository
}

// NewRiskEngine creates a new risk engine
func NewRiskEngine(config *Config, repo Repository) RiskEngine {
	return &riskEngine{
		config: config,
		repo:   repo,
	}
}

// CheckBettingLimits validates betting amount limits
func (re *riskEngine) CheckBettingLimits(userID uuid.UUID, amount decimal.Decimal, market *models.Market) error {
	// Check minimum bet amount
	minAmount := market.MinBetAmount
	if re.config.MinBetAmount.GreaterThan(minAmount) {
		minAmount = re.config.MinBetAmount
	}
	if amount.LessThan(minAmount) {
		return models.ErrBetTooSmall
	}

	// Check maximum bet amount
	maxAmount := re.config.MaxBetAmount
	if market.MaxBetAmount != nil && market.MaxBetAmount.LessThan(maxAmount) {
		maxAmount = *market.MaxBetAmount
	}
	if amount.GreaterThan(maxAmount) {
		return models.ErrBetTooLarge
	}

	// Check daily limit
	ctx := context.Background()
	dailyAmount, err := re.repo.GetUserDailyBetAmount(ctx, userID, time.Now())
	if err != nil {
		return err
	}

	if dailyAmount.Add(amount).GreaterThan(re.config.MaxDailyBetAmount) {
		return models.ErrDailyLimitExceeded
	}

	return nil
}

// CheckPositionLimits validates position size limits
func (re *riskEngine) CheckPositionLimits(userID uuid.UUID, amount decimal.Decimal, market *models.Market) error {
	if !re.config.EnablePositionLimits {
		return nil
	}

	ctx := context.Background()

	// Check current position in this market
	currentPosition, err := re.repo.GetUserPositionInMarket(ctx, userID, market.ID)
	if err != nil {
		return err
	}

	newPosition := currentPosition.Add(amount)

	// Check per-market position limit
	if newPosition.GreaterThan(re.config.MaxPositionPerMarket) {
		return models.ErrPositionLimitExceeded
	}

	// Check total position limit across all markets
	activeBets, err := re.repo.GetActiveBetsByUser(ctx, userID)
	if err != nil {
		return err
	}

	totalPosition := decimal.Zero
	for i := range activeBets {
		bet := activeBets[i]
		if bet.MarketID != market.ID { // Don't double count current market
			totalPosition = totalPosition.Add(bet.Amount)
		}
	}
	totalPosition = totalPosition.Add(newPosition) // Add new position

	if totalPosition.GreaterThan(re.config.MaxPositionPerUser) {
		return models.ErrPositionLimitExceeded
	}

	return nil
}

// CheckRateLimit validates betting rate limits
func (re *riskEngine) CheckRateLimit(userID uuid.UUID) error {
	ctx := context.Background()
	since := time.Now().Add(-time.Minute)

	betCount, err := re.repo.GetUserBetCount(ctx, userID, since)
	if err != nil {
		return err
	}

	if betCount >= re.config.MaxBetsPerMinute {
		return models.ErrRateLimitExceeded
	}

	return nil
}

// CheckCooldown validates betting cooldown period
func (re *riskEngine) CheckCooldown(userID uuid.UUID) error {
	if re.config.CooldownPeriod <= 0 {
		return nil
	}

	ctx := context.Background()
	since := time.Now().Add(-re.config.CooldownPeriod)

	betCount, err := re.repo.GetUserBetCount(ctx, userID, since)
	if err != nil {
		return err
	}

	if betCount > 0 {
		return models.ErrBetCooldownActive
	}

	return nil
}

// ValidateMarketForBetting ensures market is suitable for betting
func (re *riskEngine) ValidateMarketForBetting(market *models.Market) error {
	// Check if market is open
	if !market.CanBet() {
		return models.ErrMarketNotOpenForBetting
	}

	// Check if market closes soon (warn but don't block)
	timeToClose := time.Until(market.CloseTime)
	if timeToClose < 5*time.Minute {
		log.Printf("Warning: Market %s closes in less than 5 minutes", market.ID)
	}

	// Check if market has sufficient outcomes
	if len(market.Outcomes) < 2 {
		return models.ErrMarketNotOpenForBetting
	}

	for i := range market.Outcomes {
		outcome := market.Outcomes[i]
		if outcome.OutcomeKey == "" || outcome.OutcomeLabel == "" {
			return models.ErrMarketNotOpenForBetting
		}
	}

	return nil
}

// ValidateUserForBetting ensures user is allowed to bet
func (re *riskEngine) ValidateUserForBetting(user *models.User) error {
	// Check if user account is active
	if !*user.IsActive {
		return models.ErrUnauthorized
	}

	// Check if account is locked
	if user.IsLocked() {
		return models.ErrUnauthorized
	}

	// Check email verification
	if !user.IsEmailVerified() {
		return models.ErrUnauthorized
	}

	// Check KYC if required
	if re.config.RequireKYCForBetting && !user.IsKYCVerified() {
		return models.ErrUnauthorized
	}

	return nil
}

// CheckWalletBalance validates sufficient wallet balance
func (re *riskEngine) CheckWalletBalance(userID uuid.UUID, amount decimal.Decimal, currencyCode string) error {
	ctx := context.Background()
	wallet, err := re.repo.GetUserWallet(ctx, userID, currencyCode)
	if err != nil {
		return models.ErrInsufficientWalletBalance
	}

	if !wallet.CanDebit(amount) {
		return models.ErrInsufficientWalletBalance
	}

	return nil
}

// AssessRiskScore calculates overall risk score for a bet (0-100)
func (re *riskEngine) AssessRiskScore(userID uuid.UUID, amount decimal.Decimal, market *models.Market) (decimal.Decimal, error) {
	score := decimal.Zero
	ctx := context.Background()

	// Amount risk (25 points max)
	amountRisk := re.calculateAmountRisk(amount, market)
	score = score.Add(amountRisk.Mul(decimal.NewFromFloat(0.25)))

	// Position risk (20 points max)
	positionRisk, err := re.calculatePositionRisk(ctx, userID, amount, market)
	if err != nil {
		positionRisk = decimal.NewFromFloat(0.5) // Default medium risk if error
	}
	score = score.Add(positionRisk.Mul(decimal.NewFromFloat(0.20)))

	// Frequency risk (15 points max)
	frequencyRisk, err := re.calculateFrequencyRisk(ctx, userID)
	if err != nil {
		frequencyRisk = decimal.NewFromFloat(0.3) // Default low-medium risk
	}
	score = score.Add(frequencyRisk.Mul(decimal.NewFromFloat(0.15)))

	// Market risk (25 points max)
	marketRisk := re.calculateMarketRisk(market)
	score = score.Add(marketRisk.Mul(decimal.NewFromFloat(0.25)))

	// Time risk (15 points max)
	timeRisk := re.calculateTimeRisk(market)
	score = score.Add(timeRisk.Mul(decimal.NewFromFloat(0.15)))

	// Ensure score is within bounds
	if score.GreaterThan(decimal.NewFromInt(100)) {
		score = decimal.NewFromInt(100)
	}

	return score, nil
}

// Helper methods for risk assessment

func (re *riskEngine) calculateAmountRisk(amount decimal.Decimal, market *models.Market) decimal.Decimal {
	// Risk increases with bet size relative to limits
	maxAmount := re.config.MaxBetAmount
	if market.MaxBetAmount != nil && market.MaxBetAmount.LessThan(maxAmount) {
		maxAmount = *market.MaxBetAmount
	}

	if maxAmount.IsZero() {
		return decimal.Zero
	}

	ratio := amount.Div(maxAmount)
	if ratio.GreaterThan(decimal.NewFromInt(1)) {
		return decimal.NewFromInt(1)
	}

	return ratio
}

func (re *riskEngine) calculatePositionRisk(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, market *models.Market) (decimal.Decimal, error) {
	currentPosition, err := re.repo.GetUserPositionInMarket(ctx, userID, market.ID)
	if err != nil {
		return decimal.Zero, err
	}

	newPosition := currentPosition.Add(amount)

	// Risk based on position size relative to limit
	if re.config.MaxPositionPerMarket.IsZero() {
		return decimal.Zero, nil
	}

	ratio := newPosition.Div(re.config.MaxPositionPerMarket)
	if ratio.GreaterThan(decimal.NewFromInt(1)) {
		return decimal.NewFromInt(1), nil
	}

	return ratio, nil
}

func (re *riskEngine) calculateFrequencyRisk(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	// Check betting frequency in last hour
	since := time.Now().Add(-time.Hour)
	betCount, err := re.repo.GetUserBetCount(ctx, userID, since)
	if err != nil {
		return decimal.Zero, err
	}

	// Risk increases with frequency
	maxBetsPerHour := re.config.MaxBetsPerMinute * 60
	if maxBetsPerHour == 0 {
		return decimal.Zero, nil
	}

	ratio := decimal.NewFromInt(int64(betCount)).Div(decimal.NewFromInt(int64(maxBetsPerHour)))
	if ratio.GreaterThan(decimal.NewFromInt(1)) {
		return decimal.NewFromInt(1), nil
	}

	return ratio, nil
}

func (re *riskEngine) calculateMarketRisk(market *models.Market) decimal.Decimal {
	// Risk factors:
	// - Low liquidity
	// - Close to deadline
	// - Market imbalance

	risk := decimal.Zero

	// Liquidity risk (33% of market risk)
	if market.TotalPoolAmount.LessThan(re.config.MinBetAmount.Mul(decimal.NewFromInt(100))) {
		risk = risk.Add(decimal.NewFromFloat(0.33))
	}

	// Imbalance risk (33% of market risk)
	if len(market.Outcomes) > 1 {
		maxPool := decimal.Zero
		for i := range market.Outcomes {
			outcome := market.Outcomes[i]
			if outcome.PoolAmount.GreaterThan(maxPool) {
				maxPool = outcome.PoolAmount
			}
		}

		if market.TotalPoolAmount.GreaterThan(decimal.Zero) {
			imbalance := maxPool.Div(market.TotalPoolAmount)
			if imbalance.GreaterThan(decimal.NewFromFloat(0.8)) {
				risk = risk.Add(decimal.NewFromFloat(0.33))
			}
		}
	}

	// Status risk (34% of market risk)
	if market.Status != models.MarketStatusOpen {
		risk = risk.Add(decimal.NewFromFloat(0.34))
	}

	return risk
}

func (re *riskEngine) calculateTimeRisk(market *models.Market) decimal.Decimal {
	timeToClose := time.Until(market.CloseTime)

	// High risk if less than 1 hour to close
	if timeToClose < time.Hour {
		return decimal.NewFromInt(1)
	}

	// Medium risk if less than 24 hours to close
	if timeToClose < 24*time.Hour {
		return decimal.NewFromFloat(0.5)
	}

	return decimal.Zero
}
