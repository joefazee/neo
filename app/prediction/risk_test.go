package prediction

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/joefazee/neo/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of the Repository interface for testing.
type MockRepository struct {
	mock.Mock
}

// WithTx allows creating a transaction-scoped mock repository.
// In tests, this can be used to set expectations on a new mock instance
// that is passed into the transactional block of code.
func (m *MockRepository) WithTx(tx *gorm.DB) Repository {
	args := m.Called(tx)
	// Return a new instance of the mock repository that the service will use within the transaction.
	// This allows setting separate expectations for transactional operations.
	return args.Get(0).(Repository)
}

func (m *MockRepository) GetBetByID(ctx context.Context, id uuid.UUID) (*models.Bet, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Bet), args.Error(1)
}

func (m *MockRepository) GetBetsByUser(ctx context.Context, userID uuid.UUID, filters *BetFilters) ([]models.Bet, int64, error) {
	args := m.Called(ctx, userID, filters)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.Bet), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) GetBetsByMarket(ctx context.Context, marketID uuid.UUID) ([]models.Bet, error) {
	args := m.Called(ctx, marketID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Bet), args.Error(1)
}

func (m *MockRepository) GetActiveBetsByUser(ctx context.Context, userID uuid.UUID) ([]models.Bet, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Bet), args.Error(1)
}

func (m *MockRepository) CreateBet(ctx context.Context, bet *models.Bet) error {
	args := m.Called(ctx, bet)
	return args.Error(0)
}

func (m *MockRepository) UpdateBet(ctx context.Context, bet *models.Bet) error {
	args := m.Called(ctx, bet)
	return args.Error(0)
}

func (m *MockRepository) GetUserPositionInMarket(ctx context.Context, userID, marketID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, userID, marketID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockRepository) GetUserDailyBetAmount(ctx context.Context, userID uuid.UUID, date time.Time) (decimal.Decimal, error) {
	args := m.Called(ctx, userID, date)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockRepository) GetUserBetCount(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	args := m.Called(ctx, userID, since)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetMarketWithOutcomes(ctx context.Context, marketID uuid.UUID) (*models.Market, error) {
	args := m.Called(ctx, marketID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Market), args.Error(1)
}

func (m *MockRepository) GetMarketOutcome(ctx context.Context, outcomeID uuid.UUID) (*models.MarketOutcome, error) {
	args := m.Called(ctx, outcomeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MarketOutcome), args.Error(1)
}

func (m *MockRepository) UpdateMarketOutcome(ctx context.Context, outcome *models.MarketOutcome) error {
	args := m.Called(ctx, outcome)
	return args.Error(0)
}

func (m *MockRepository) UpdateMarket(ctx context.Context, market *models.Market) error {
	args := m.Called(ctx, market)
	return args.Error(0)
}

func (m *MockRepository) GetUserWallet(ctx context.Context, userID uuid.UUID, currencyCode string) (*models.Wallet, error) {
	args := m.Called(ctx, userID, currencyCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockRepository) UpdateWallet(ctx context.Context, wallet *models.Wallet) error {
	args := m.Called(ctx, wallet)
	return args.Error(0)
}

func (m *MockRepository) CreateTransaction(ctx context.Context, transaction *models.Transaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

// UpdateTransaction is the newly added method to satisfy the interface.
func (m *MockRepository) UpdateTransaction(ctx context.Context, transaction *models.Transaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

func TestRiskEngine_CheckBettingLimits(t *testing.T) {
	config := GetDefaultConfig() // Use your actual default config
	mockRepo := new(MockRepository)

	userID := uuid.New()
	market := &models.Market{
		MinBetAmount: config.MinBetAmount.Sub(decimal.NewFromInt(10)),                          // Market min is lower than global
		MaxBetAmount: &[]decimal.Decimal{config.MaxBetAmount.Add(decimal.NewFromInt(1000))}[0], // Market max is higher
	}

	// General mock for daily amount, can be overridden in specific sub-tests if needed
	mockRepo.On("GetUserDailyBetAmount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(decimal.NewFromInt(500), nil).Maybe()

	t.Run("Valid bet amount", func(t *testing.T) {
		// Specific mock for this sub-test if the general one is too broad or causes issues
		localMockRepo := new(MockRepository)
		localEngine := NewRiskEngine(config, localMockRepo)
		localMockRepo.On("GetUserDailyBetAmount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(decimal.NewFromInt(500), nil).Once()

		err := localEngine.CheckBettingLimits(userID, config.MinBetAmount.Add(decimal.NewFromInt(1)), market)
		assert.NoError(t, err)
		localMockRepo.AssertExpectations(t)
	})

	t.Run("Bet too small (global config)", func(t *testing.T) {
		localMockRepo := new(MockRepository)
		localEngine := NewRiskEngine(config, localMockRepo)
		// GetUserDailyBetAmount won't be called if amount checks fail first
		// No mock needed here for GetUserDailyBetAmount unless the logic changes

		err := localEngine.CheckBettingLimits(userID, config.MinBetAmount.Sub(decimal.NewFromInt(1)), market)
		assert.EqualError(t, err, models.ErrBetTooSmall.Error())
	})

	t.Run("Bet too small (market specific, lower than global)", func(t *testing.T) {
		localMockRepo := new(MockRepository)
		tempConfig := *config // Make a copy to modify MinBetAmount locally for this test
		localEngine := NewRiskEngine(&tempConfig, localMockRepo)

		marketSpecificMin := decimal.NewFromInt(50)
		market.MinBetAmount = marketSpecificMin
		tempConfig.MinBetAmount = decimal.NewFromInt(100) // Global is higher

		err := localEngine.CheckBettingLimits(userID, marketSpecificMin.Sub(decimal.NewFromInt(1)), market)
		assert.EqualError(t, err, models.ErrBetTooSmall.Error()) // Should fail against global min
	})

	t.Run("Bet too large (global config)", func(t *testing.T) {
		localMockRepo := new(MockRepository)
		localEngine := NewRiskEngine(config, localMockRepo)

		err := localEngine.CheckBettingLimits(userID, config.MaxBetAmount.Add(decimal.NewFromInt(1)), market)
		assert.EqualError(t, err, models.ErrBetTooLarge.Error())
	})

	t.Run("Bet too large (market specific, higher than global)", func(t *testing.T) {
		localMockRepo := new(MockRepository)
		tempConfig := *config
		localEngine := NewRiskEngine(&tempConfig, localMockRepo)

		marketSpecificMax := decimal.NewFromInt(100000)
		market.MaxBetAmount = &marketSpecificMax
		tempConfig.MaxBetAmount = decimal.NewFromInt(50000) // Global is lower

		err := localEngine.CheckBettingLimits(userID, marketSpecificMax.Add(decimal.NewFromInt(1)), market)
		assert.EqualError(t, err, models.ErrBetTooLarge.Error()) // Should fail against global max
	})

	t.Run("Exceeds daily limit", func(t *testing.T) {
		localMockRepo := new(MockRepository)
		localEngine := NewRiskEngine(config, localMockRepo)
		localMockRepo.On("GetUserDailyBetAmount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(config.MaxDailyBetAmount.Sub(decimal.NewFromInt(50)), nil).Once()

		err := localEngine.CheckBettingLimits(userID, decimal.NewFromInt(100), market)
		assert.EqualError(t, err, models.ErrDailyLimitExceeded.Error())
		localMockRepo.AssertExpectations(t)
	})

	t.Run("Repo error for daily amount", func(t *testing.T) {
		localMockRepo := new(MockRepository)
		localEngine := NewRiskEngine(config, localMockRepo)
		localMockRepo.On("GetUserDailyBetAmount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(decimal.Zero, errors.New("db error")).Once()

		err := localEngine.CheckBettingLimits(userID, decimal.NewFromInt(100), market)
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		localMockRepo.AssertExpectations(t)
	})
}

func TestRiskEngine_CheckPositionLimits(t *testing.T) {
	config := GetDefaultConfig()
	config.EnablePositionLimits = true // Ensure it's enabled for these tests

	userID := uuid.New()
	marketID := uuid.New()
	otherMarketID := uuid.New()
	market := &models.Market{ID: marketID}

	t.Run("Position limit disabled", func(t *testing.T) {
		tempConfig := *config
		tempConfig.EnablePositionLimits = false
		mockRepo := new(MockRepository) // Fresh mock
		tempEngine := NewRiskEngine(&tempConfig, mockRepo)
		err := tempEngine.CheckPositionLimits(userID, decimal.NewFromInt(1000), market)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t) // Should not have made any calls
	})

	t.Run("Valid position", func(t *testing.T) {
		mockRepo := new(MockRepository) // Fresh mock
		engine := NewRiskEngine(config, mockRepo)
		mockRepo.On("GetUserPositionInMarket", mock.Anything, userID, marketID).Return(decimal.NewFromInt(1000), nil).Once()
		mockRepo.On("GetActiveBetsByUser", mock.Anything, userID).Return([]models.Bet{
			{MarketID: otherMarketID, Amount: decimal.NewFromInt(2000)},
		}, nil).Once()
		err := engine.CheckPositionLimits(userID, decimal.NewFromInt(500), market) // New position = 1500, Total = 1500+2000=3500
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Exceeds per-market limit", func(t *testing.T) {
		mockRepo := new(MockRepository) // Fresh mock
		engine := NewRiskEngine(config, mockRepo)
		mockRepo.On("GetUserPositionInMarket", mock.Anything, userID, marketID).Return(config.MaxPositionPerMarket.Sub(decimal.NewFromInt(100)), nil).Once()
		err := engine.CheckPositionLimits(userID, decimal.NewFromInt(200), market)
		assert.EqualError(t, err, models.ErrPositionLimitExceeded.Error())
		mockRepo.AssertExpectations(t)
	})

	t.Run("Exceeds total user limit", func(t *testing.T) {
		mockRepo := new(MockRepository) // Fresh mock
		engine := NewRiskEngine(config, mockRepo)
		mockRepo.On("GetUserPositionInMarket", mock.Anything, userID, marketID).Return(decimal.NewFromInt(1000), nil).Once()
		mockRepo.On("GetActiveBetsByUser", mock.Anything, userID).Return([]models.Bet{
			{MarketID: otherMarketID, Amount: config.MaxPositionPerUser.Sub(decimal.NewFromInt(1500))},
		}, nil).Once()
		err := engine.CheckPositionLimits(userID, decimal.NewFromInt(501), market)
		assert.EqualError(t, err, models.ErrPositionLimitExceeded.Error())
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repo error GetUserPositionInMarket", func(t *testing.T) {
		mockRepo := new(MockRepository) // Fresh mock
		engine := NewRiskEngine(config, mockRepo)
		mockRepo.On("GetUserPositionInMarket", mock.Anything, userID, marketID).Return(decimal.Zero, errors.New("db error")).Once()
		err := engine.CheckPositionLimits(userID, decimal.NewFromInt(100), market)
		assert.ErrorContains(t, err, "db error")
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repo error GetActiveBetsByUser", func(t *testing.T) {
		mockRepo := new(MockRepository) // Fresh mock
		engine := NewRiskEngine(config, mockRepo)
		mockRepo.On("GetUserPositionInMarket", mock.Anything, userID, marketID).Return(decimal.Zero, nil).Once()
		mockRepo.On("GetActiveBetsByUser", mock.Anything, userID).Return(nil, errors.New("db error")).Once()
		err := engine.CheckPositionLimits(userID, decimal.NewFromInt(100), market)
		assert.ErrorContains(t, err, "db error")
		mockRepo.AssertExpectations(t)
	})
}

func TestRiskEngine_CheckRateLimit(t *testing.T) {
	config := GetDefaultConfig()
	userID := uuid.New()

	t.Run("Within rate limit", func(t *testing.T) {
		mockRepo := new(MockRepository) // Fresh mock
		engine := NewRiskEngine(config, mockRepo)
		mockRepo.On("GetUserBetCount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(config.MaxBetsPerMinute-1, nil).Once()
		err := engine.CheckRateLimit(userID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Exceeds rate limit", func(t *testing.T) {
		mockRepo := new(MockRepository) // Fresh mock
		engine := NewRiskEngine(config, mockRepo)
		mockRepo.On("GetUserBetCount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(config.MaxBetsPerMinute, nil).Once()
		err := engine.CheckRateLimit(userID)
		assert.EqualError(t, err, models.ErrRateLimitExceeded.Error())
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repo error", func(t *testing.T) {
		mockRepo := new(MockRepository) // Fresh mock
		engine := NewRiskEngine(config, mockRepo)
		mockRepo.On("GetUserBetCount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(0, errors.New("db error")).Once()
		err := engine.CheckRateLimit(userID)
		assert.ErrorContains(t, err, "db error")
		mockRepo.AssertExpectations(t)
	})
}

func TestRiskEngine_CheckCooldown(t *testing.T) {
	config := GetDefaultConfig()
	config.CooldownPeriod = 5 * time.Second // Set a specific cooldown for test
	userID := uuid.New()

	t.Run("Cooldown period not active", func(t *testing.T) {
		mockRepo := new(MockRepository) // Fresh mock
		engine := NewRiskEngine(config, mockRepo)
		mockRepo.On("GetUserBetCount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(0, nil).Once()
		err := engine.CheckCooldown(userID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Cooldown period active", func(t *testing.T) {
		mockRepo := new(MockRepository) // Fresh mock
		engine := NewRiskEngine(config, mockRepo)
		mockRepo.On("GetUserBetCount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(1, nil).Once()
		err := engine.CheckCooldown(userID)
		assert.EqualError(t, err, models.ErrBetCooldownActive.Error())
		mockRepo.AssertExpectations(t)
	})

	t.Run("Cooldown disabled in config", func(t *testing.T) {
		localMockRepo := new(MockRepository) // Use a local mock for AssertNotCalled
		tempConfig := *config
		tempConfig.CooldownPeriod = 0
		tempEngine := NewRiskEngine(&tempConfig, localMockRepo) // Pass the local mock

		err := tempEngine.CheckCooldown(userID)
		assert.NoError(t, err)
		// AssertNotCalled applies to localMockRepo, which hasn't had GetUserBetCount called on it.
		localMockRepo.AssertNotCalled(t, "GetUserBetCount", mock.Anything, userID, mock.AnythingOfType("time.Time"))
	})

	t.Run("Repo error", func(t *testing.T) {
		mockRepo := new(MockRepository) // Fresh mock
		engine := NewRiskEngine(config, mockRepo)
		mockRepo.On("GetUserBetCount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(0, errors.New("db error")).Once()
		err := engine.CheckCooldown(userID)
		assert.ErrorContains(t, err, "db error")
		mockRepo.AssertExpectations(t)
	})
}

func TestRiskEngine_ValidateMarketForBetting(t *testing.T) {
	engine := NewRiskEngine(GetDefaultConfig(), new(MockRepository))
	marketID := uuid.New()

	t.Run("Valid market", func(t *testing.T) {
		market := &models.Market{
			ID:        marketID,
			Status:    models.MarketStatusOpen,
			CloseTime: time.Now().Add(10 * time.Minute),
			Outcomes: []models.MarketOutcome{
				{OutcomeKey: "yes", OutcomeLabel: "Yes"},
				{OutcomeKey: "no", OutcomeLabel: "No"},
			},
		}
		err := engine.ValidateMarketForBetting(market)
		assert.NoError(t, err)
	})

	t.Run("Market not open", func(t *testing.T) {
		market := &models.Market{Status: models.MarketStatusClosed, CloseTime: time.Now().Add(time.Hour)}
		err := engine.ValidateMarketForBetting(market)
		assert.EqualError(t, err, models.ErrMarketNotOpenForBetting.Error())
	})

	t.Run("Market closed (past CloseTime)", func(t *testing.T) {
		market := &models.Market{Status: models.MarketStatusOpen, CloseTime: time.Now().Add(-time.Hour)}
		err := engine.ValidateMarketForBetting(market)
		assert.EqualError(t, err, models.ErrMarketNotOpenForBetting.Error())
	})

	t.Run("Not enough outcomes", func(t *testing.T) {
		market := &models.Market{
			Status:    models.MarketStatusOpen,
			CloseTime: time.Now().Add(10 * time.Minute),
			Outcomes:  []models.MarketOutcome{{OutcomeKey: "yes", OutcomeLabel: "Yes"}},
		}
		err := engine.ValidateMarketForBetting(market)
		assert.EqualError(t, err, models.ErrMarketNotOpenForBetting.Error())
	})

	t.Run("Outcome missing key", func(t *testing.T) {
		market := &models.Market{
			Status:    models.MarketStatusOpen,
			CloseTime: time.Now().Add(10 * time.Minute),
			Outcomes: []models.MarketOutcome{
				{OutcomeLabel: "Yes"},
				{OutcomeKey: "no", OutcomeLabel: "No"},
			},
		}
		err := engine.ValidateMarketForBetting(market)
		assert.EqualError(t, err, models.ErrMarketNotOpenForBetting.Error())
	})
}

func TestRiskEngine_ValidateUserForBetting(t *testing.T) {
	config := GetDefaultConfig()
	engine := NewRiskEngine(config, new(MockRepository))
	now := time.Now()
	isActive := true
	isNotActive := false

	t.Run("Valid user", func(t *testing.T) {
		user := &models.User{IsActive: &isActive, EmailVerifiedAt: &now, KYCStatus: models.KYCStatusVerified, KYCVerifiedAt: &now}
		config.RequireKYCForBetting = true
		err := engine.ValidateUserForBetting(user)
		assert.NoError(t, err)
	})

	t.Run("User not active", func(t *testing.T) {
		user := &models.User{IsActive: &isNotActive}
		err := engine.ValidateUserForBetting(user)
		assert.EqualError(t, err, models.ErrUnauthorized.Error())
	})

	t.Run("User locked", func(t *testing.T) {
		lockedUntil := time.Now().Add(time.Hour)
		user := &models.User{IsActive: &isActive, LockedUntil: &lockedUntil}
		err := engine.ValidateUserForBetting(user)
		assert.EqualError(t, err, models.ErrUnauthorized.Error())
	})

	t.Run("Email not verified", func(t *testing.T) {
		user := &models.User{IsActive: &isActive, EmailVerifiedAt: nil}
		err := engine.ValidateUserForBetting(user)
		assert.EqualError(t, err, models.ErrUnauthorized.Error())
	})

	t.Run("KYC required and not verified", func(t *testing.T) {
		config.RequireKYCForBetting = true
		user := &models.User{IsActive: &isActive, EmailVerifiedAt: &now, KYCStatus: models.KYCStatusPending}
		err := engine.ValidateUserForBetting(user)
		assert.EqualError(t, err, models.ErrUnauthorized.Error())
	})

	t.Run("KYC not required, user not KYC verified", func(t *testing.T) {
		config.RequireKYCForBetting = false
		user := &models.User{IsActive: &isActive, EmailVerifiedAt: &now, KYCStatus: models.KYCStatusPending}
		err := engine.ValidateUserForBetting(user)
		assert.NoError(t, err)
	})
}

func TestRiskEngine_CheckWalletBalance(t *testing.T) {
	userID := uuid.New()
	currency := "USD"

	t.Run("Sufficient balance", func(t *testing.T) {
		mockRepo := new(MockRepository) // Fresh mock
		engine := NewRiskEngine(GetDefaultConfig(), mockRepo)
		wallet := &models.Wallet{Balance: decimal.NewFromInt(1000), LockedBalance: decimal.NewFromInt(100)} // Available 900
		mockRepo.On("GetUserWallet", mock.Anything, userID, currency).Return(wallet, nil).Once()
		err := engine.CheckWalletBalance(userID, decimal.NewFromInt(500), currency)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Insufficient balance", func(t *testing.T) {
		mockRepo := new(MockRepository) // Fresh mock
		engine := NewRiskEngine(GetDefaultConfig(), mockRepo)
		wallet := &models.Wallet{Balance: decimal.NewFromInt(500), LockedBalance: decimal.NewFromInt(100)} // Available 400
		mockRepo.On("GetUserWallet", mock.Anything, userID, currency).Return(wallet, nil).Once()
		err := engine.CheckWalletBalance(userID, decimal.NewFromInt(500), currency)
		assert.EqualError(t, err, models.ErrInsufficientWalletBalance.Error())
		mockRepo.AssertExpectations(t)
	})

	t.Run("Wallet not found", func(t *testing.T) {
		mockRepo := new(MockRepository) // Fresh mock
		engine := NewRiskEngine(GetDefaultConfig(), mockRepo)
		mockRepo.On("GetUserWallet", mock.Anything, userID, currency).Return(nil, errors.New("wallet not found")).Once()
		err := engine.CheckWalletBalance(userID, decimal.NewFromInt(100), currency)
		assert.EqualError(t, err, models.ErrInsufficientWalletBalance.Error()) // Should interpret as insufficient
		mockRepo.AssertExpectations(t)
	})
}

func TestRiskEngine_AssessRiskScore(t *testing.T) {
	config := GetDefaultConfig()
	userID := uuid.New()
	marketID := uuid.New()

	baseMarket := &models.Market{
		ID:              marketID,
		MinBetAmount:    config.MinBetAmount,
		MaxBetAmount:    &config.MaxBetAmount,
		TotalPoolAmount: decimal.NewFromInt(20000),
		Status:          models.MarketStatusOpen,
		CloseTime:       time.Now().Add(48 * time.Hour),
		Outcomes: []models.MarketOutcome{
			{PoolAmount: decimal.NewFromInt(10000)},
			{PoolAmount: decimal.NewFromInt(10000)},
		},
	}
	baseAmount := config.MinBetAmount.Add(decimal.NewFromInt(1))

	// --- Baseline Low Risk Score Calculation ---
	// 1. Amount Risk Component (arc)
	//    ratio = 101 / 50000 = 0.00202
	//    arc_weighted = 0.00202 * 0.25 = 0.000505
	lowAmountRiskRatio := baseAmount.Div(config.MaxBetAmount)
	lowAmountComponent := lowAmountRiskRatio.Mul(decimal.NewFromFloat(0.25))

	// 2. Position Risk Component (prc) - assuming current position 0, new position = 101
	//    ratio = 101 / 100000 (MaxPositionPerMarket) = 0.00101
	//    prc_weighted = 0.00101 * 0.20 = 0.000202
	lowPositionRiskRatio := baseAmount.Div(config.MaxPositionPerMarket) // newPosition is baseAmount as current is 0
	lowPositionComponent := lowPositionRiskRatio.Mul(decimal.NewFromFloat(0.20))

	// 3. Frequency Risk Component (frc) - assuming 0 bets in last hour
	//    ratio = 0 / (10*60) = 0
	//    frc_weighted = 0 * 0.15 = 0
	lowFrequencyComponent := decimal.Zero

	// 4. Market Risk Component (mrc)
	//    Liquidity: 20000 < (100*100=10000) is false. Risk = 0.
	//    Imbalance: maxPool=10000, totalPool=20000. imbalance = 0.5. 0.5 > 0.8 is false. Risk = 0.
	//    Status: Open. Risk = 0.
	//    mrc_raw = 0. mrc_weighted = 0 * 0.25 = 0
	lowMarketComponent := decimal.Zero

	// 5. Time Risk Component (trc)
	//    CloseTime = 48h away. Not <1h, not <24h. Risk_raw = 0.
	//    trc_weighted = 0 * 0.15 = 0
	lowTimeComponent := decimal.Zero

	expectedLowRiskScore := lowAmountComponent.Add(lowPositionComponent).Add(lowFrequencyComponent).Add(lowMarketComponent).Add(lowTimeComponent)
	// expectedLowRiskScore = 0.000505 + 0.000202 + 0 + 0 + 0 = 0.000707

	t.Run("Low risk scenario", func(t *testing.T) {
		mockRepo := new(MockRepository)
		engine := NewRiskEngine(config, mockRepo)
		mockRepo.On("GetUserPositionInMarket", mock.Anything, userID, marketID).Return(decimal.Zero, nil).Once()
		mockRepo.On("GetUserBetCount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(0, nil).Once()

		score, err := engine.AssessRiskScore(userID, baseAmount, baseMarket)
		assert.NoError(t, err)
		assert.True(t, score.Equal(expectedLowRiskScore), "Expected low risk score %s, got %s", expectedLowRiskScore, score)
		mockRepo.AssertExpectations(t)
	})

	t.Run("High amount risk", func(t *testing.T) {
		mockRepo := new(MockRepository)
		engine := NewRiskEngine(config, mockRepo)
		mockRepo.On("GetUserPositionInMarket", mock.Anything, userID, marketID).Return(decimal.Zero, nil).Once()
		mockRepo.On("GetUserBetCount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(0, nil).Once()

		highAmount := config.MaxBetAmount.Mul(decimal.NewFromFloat(0.9)) // 45000

		// Recalculate components with highAmount
		// 1. Amount Risk
		highAmountRiskRatio := highAmount.Div(config.MaxBetAmount)
		highAmountComponent := highAmountRiskRatio.Mul(decimal.NewFromFloat(0.25))

		// 2. Position Risk (influenced by highAmount)
		//    currentPosition = 0, newPosition = highAmount (45000)
		//    ratio = 45000 / 100000 = 0.45
		//    prc_weighted = 0.45 * 0.20 = 0.09
		currentHighAmountPositionRiskRatio := highAmount.Div(config.MaxPositionPerMarket)
		currentHighAmountPositionComponent := currentHighAmountPositionRiskRatio.Mul(decimal.NewFromFloat(0.20))

		expectedScore := highAmountComponent.Add(currentHighAmountPositionComponent).Add(lowFrequencyComponent).Add(lowMarketComponent).Add(lowTimeComponent)
		// expectedScore = 0.225 + 0.09 + 0 + 0 + 0 = 0.315

		score, err := engine.AssessRiskScore(userID, highAmount, baseMarket)
		assert.NoError(t, err)
		assert.True(t, score.Equal(expectedScore), "Expected score %s due to amount, got %s", expectedScore, score)
		mockRepo.AssertExpectations(t)
	})

	t.Run("High position risk", func(t *testing.T) {
		mockRepo := new(MockRepository)
		engine := NewRiskEngine(config, mockRepo)

		currentPos := config.MaxPositionPerMarket.Mul(decimal.NewFromFloat(0.8))           // 80000
		betAmountForHighPos := config.MaxPositionPerMarket.Mul(decimal.NewFromFloat(0.15)) // 15000
		// newPosition = 80000 + 15000 = 95000
		// positionRiskRatio = 95000 / 100000 = 0.95
		// positionComponent = 0.95 * 0.20 = 0.19
		highPositionRiskRatio := currentPos.Add(betAmountForHighPos).Div(config.MaxPositionPerMarket)
		highPositionComponent := highPositionRiskRatio.Mul(decimal.NewFromFloat(0.20))

		// Amount risk for betAmountForHighPos
		// amountRiskRatio = 15000 / 50000 = 0.3
		// amountComponent = 0.3 * 0.25 = 0.075
		currentBetAmountRiskRatio := betAmountForHighPos.Div(config.MaxBetAmount)
		currentBetAmountComponent := currentBetAmountRiskRatio.Mul(decimal.NewFromFloat(0.25))

		expectedScore := currentBetAmountComponent.Add(highPositionComponent).Add(lowFrequencyComponent).Add(lowMarketComponent).Add(lowTimeComponent)
		// expectedScore = 0.075 + 0.19 + 0 + 0 + 0 = 0.265

		mockRepo.On("GetUserPositionInMarket", mock.Anything, userID, marketID).Return(currentPos, nil).Once()
		mockRepo.On("GetUserBetCount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(0, nil).Once()

		score, err := engine.AssessRiskScore(userID, betAmountForHighPos, baseMarket)
		assert.NoError(t, err)
		assert.True(t, score.Equal(expectedScore), "Expected score %s due to position, got %s", expectedScore, score)
		mockRepo.AssertExpectations(t)
	})

	t.Run("High frequency risk", func(t *testing.T) {
		mockRepo := new(MockRepository)
		engine := NewRiskEngine(config, mockRepo)

		// frequencyRiskRatio = (10*50) / (10*60) = 500 / 600 = 5/6
		// frequencyComponent = (5/6) * 0.15 = 0.125
		highFrequencyRiskRatio := decimal.NewFromInt(int64(config.MaxBetsPerMinute * 50)).Div(decimal.NewFromInt(int64(config.MaxBetsPerMinute * 60)))
		highFrequencyComponent := highFrequencyRiskRatio.Mul(decimal.NewFromFloat(0.15))

		expectedScore := lowAmountComponent.Add(lowPositionComponent).Add(highFrequencyComponent).Add(lowMarketComponent).Add(lowTimeComponent)
		// expectedScore = 0.000505 + 0.000202 + 0.125 = 0.125707

		mockRepo.On("GetUserPositionInMarket", mock.Anything, userID, marketID).Return(decimal.Zero, nil).Once()
		mockRepo.On("GetUserBetCount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(config.MaxBetsPerMinute*50, nil).Once()

		score, err := engine.AssessRiskScore(userID, baseAmount, baseMarket)
		assert.NoError(t, err)
		assert.True(t, score.Equal(expectedScore), "Expected score %s due to frequency, got %s", expectedScore, score)
		mockRepo.AssertExpectations(t)
	})

	t.Run("High market risk (low liquidity)", func(t *testing.T) {
		mockRepo := new(MockRepository)
		engine := NewRiskEngine(config, mockRepo)
		mockRepo.On("GetUserPositionInMarket", mock.Anything, userID, marketID).Return(decimal.Zero, nil).Once()
		mockRepo.On("GetUserBetCount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(0, nil).Once()

		illiquidMarket := *baseMarket
		illiquidMarket.TotalPoolAmount = config.MinBetAmount.Mul(decimal.NewFromInt(10)) // 1000

		if len(illiquidMarket.Outcomes) > 0 { // Ensure outcomes exist before trying to modify
			numOutcomes := decimal.NewFromInt(int64(len(illiquidMarket.Outcomes)))
			if numOutcomes.IsZero() { // Should not happen if baseMarket is valid
				numOutcomes = decimal.NewFromInt(1) // Avoid division by zero
			}
			balancedPoolPerOutcome := illiquidMarket.TotalPoolAmount.Div(numOutcomes)

			newOutcomes := make([]models.MarketOutcome, len(illiquidMarket.Outcomes))
			for i := range illiquidMarket.Outcomes {
				newOutcomes[i] = models.MarketOutcome{ // Create new instances
					// Copy other relevant fields if necessary, e.g., ID, OutcomeKey
					ID:           illiquidMarket.Outcomes[i].ID,
					OutcomeKey:   illiquidMarket.Outcomes[i].OutcomeKey,
					OutcomeLabel: illiquidMarket.Outcomes[i].OutcomeLabel,
					PoolAmount:   balancedPoolPerOutcome,
				}
			}
			illiquidMarket.Outcomes = newOutcomes
		}

		marketRiskRaw := decimal.NewFromFloat(0.33) // Only liquidity part triggered
		highMarketComponent := marketRiskRaw.Mul(decimal.NewFromFloat(0.25))
		expectedScore := lowAmountComponent.Add(lowPositionComponent).Add(lowFrequencyComponent).Add(highMarketComponent).Add(lowTimeComponent)
		// expectedScore = 0.000505 + 0.000202 + 0 + 0.0825 + 0 = 0.083207

		score, err := engine.AssessRiskScore(userID, baseAmount, &illiquidMarket)
		assert.NoError(t, err)
		assert.True(t, score.Equal(expectedScore), "Expected score %s due to market liquidity, got %s", expectedScore, score)
		mockRepo.AssertExpectations(t)
	})

	t.Run("High time risk (closing soon)", func(t *testing.T) {
		mockRepo := new(MockRepository)
		engine := NewRiskEngine(config, mockRepo)
		mockRepo.On("GetUserPositionInMarket", mock.Anything, userID, marketID).Return(decimal.Zero, nil).Once()
		mockRepo.On("GetUserBetCount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(0, nil).Once()

		closingSoonMarket := *baseMarket
		closingSoonMarket.CloseTime = time.Now().Add(30 * time.Minute)
		// timeRisk_raw = 1
		// timeComponent = 1 * 0.15 = 0.15
		highTimeComponent := decimal.NewFromInt(1).Mul(decimal.NewFromFloat(0.15))
		expectedScore := lowAmountComponent.Add(lowPositionComponent).Add(lowFrequencyComponent).Add(lowMarketComponent).Add(highTimeComponent)
		// expectedScore = 0.000505 + 0.000202 + 0 + 0 + 0.15 = 0.150707

		score, err := engine.AssessRiskScore(userID, baseAmount, &closingSoonMarket)
		assert.NoError(t, err)
		assert.True(t, score.Equal(expectedScore), "Expected score %s due to time, got %s", expectedScore, score)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error in GetUserPositionInMarket", func(t *testing.T) {
		mockRepo := new(MockRepository)
		engine := NewRiskEngine(config, mockRepo)
		mockRepo.On("GetUserPositionInMarket", mock.Anything, userID, marketID).Return(decimal.Zero, errors.New("db error")).Once()
		mockRepo.On("GetUserBetCount", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(0, nil).Once()

		// positionRisk defaults to 0.5 if error.
		// positionComponent = 0.5 * 0.20 = 0.10
		defaultErrorPositionComponent := decimal.NewFromFloat(0.5).Mul(decimal.NewFromFloat(0.20))
		expectedScore := lowAmountComponent.Add(defaultErrorPositionComponent).Add(lowFrequencyComponent).Add(lowMarketComponent).Add(lowTimeComponent)
		// expectedScore = 0.000505 + 0.10 + 0 + 0 + 0 = 0.100505

		score, err := engine.AssessRiskScore(userID, baseAmount, baseMarket)
		assert.NoError(t, err)
		assert.True(t, score.Equal(expectedScore), "Expected score %s with default medium position risk, got %s", expectedScore, score)
		mockRepo.AssertExpectations(t)
	})
}
