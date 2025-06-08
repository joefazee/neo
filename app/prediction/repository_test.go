package prediction

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"

	"github.com/joefazee/neo/models"
	"github.com/joefazee/neo/tests/suites"
)

type PredictionRepositoryTestSuite struct {
	suites.RepositoryTestSuite
	repo Repository
}

func (suite *PredictionRepositoryTestSuite) SetupSuite() {
	if testing.Short() {
		suite.T().Skip("Skipping database integration test")
	}

	suite.AutoMigrate = true // Ensure migrations run if this is a fresh DB

	suite.RepositoryTestSuite.SetupSuite()

	suite.repo = NewRepository(suite.DB)
}

func TestPredictionRepository(t *testing.T) {
	suite.Run(t, new(PredictionRepositoryTestSuite))
}
func (suite *PredictionRepositoryTestSuite) TestGetBetByID() {
	ctx := context.Background()
	bet := suite.createTestBet()

	result, err := suite.repo.GetBetByID(ctx, bet.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Equal(bet.ID, result.ID)
	suite.Assert().NotNil(result.Market)
	suite.Assert().NotNil(result.MarketOutcome)
	suite.Assert().NotNil(result.User)
}

func (suite *PredictionRepositoryTestSuite) TestGetBetByID_NotFound() {
	ctx := context.Background()

	result, err := suite.repo.GetBetByID(ctx, uuid.New())
	suite.AssertDBError(err)
	suite.Assert().Nil(result)
	suite.Assert().ErrorIs(err, gorm.ErrRecordNotFound)
}

func (suite *PredictionRepositoryTestSuite) TestGetBetsByUser() {
	ctx := context.Background()
	user := suite.createTestUser()

	// Create multiple bets for the user
	bet1 := suite.createTestBetForUser(user.ID)
	bet2 := suite.createTestBetForUser(user.ID)

	// Create bet for different user (should not be returned)
	otherUser := suite.createTestUser()
	suite.createTestBetForUser(otherUser.ID)

	filters := &BetFilters{
		Page:    1,
		PerPage: 10,
	}

	bets, total, err := suite.repo.GetBetsByUser(ctx, user.ID, filters)
	suite.AssertNoDBError(err)
	suite.Assert().Len(bets, 2)
	suite.Assert().Equal(int64(2), total)

	// Verify returned bets belong to correct user
	betIDs := []uuid.UUID{bets[0].ID, bets[1].ID}
	suite.Assert().Contains(betIDs, bet1.ID)
	suite.Assert().Contains(betIDs, bet2.ID)
}

func (suite *PredictionRepositoryTestSuite) TestGetBetsByUser_WithFilters() {
	ctx := context.Background()
	user := suite.createTestUser()
	market := suite.createTestMarket()

	// Create bets with different amounts
	bet1 := suite.createTestBetWithAmount(user.ID, market.ID, decimal.NewFromFloat(100))
	suite.createTestBetWithAmount(user.ID, market.ID, decimal.NewFromFloat(200))

	minAmount := decimal.NewFromFloat(150)
	filters := &BetFilters{
		MinAmount: &minAmount,
		Page:      1,
		PerPage:   10,
	}

	bets, total, err := suite.repo.GetBetsByUser(ctx, user.ID, filters)
	suite.AssertNoDBError(err)
	suite.Assert().Len(bets, 1)
	suite.Assert().Equal(int64(1), total)
	suite.Assert().NotEqual(bet1.ID, bets[0].ID) // Should be the 200 amount bet
}

func (suite *PredictionRepositoryTestSuite) TestGetBetsByUser_Pagination() {
	ctx := context.Background()
	user := suite.createTestUser()

	// Create 5 bets
	for i := 0; i < 5; i++ {
		suite.createTestBetForUser(user.ID)
	}

	// Test first page
	filters := &BetFilters{Page: 1, PerPage: 2}
	bets, total, err := suite.repo.GetBetsByUser(ctx, user.ID, filters)
	suite.AssertNoDBError(err)
	suite.Assert().Len(bets, 2)
	suite.Assert().Equal(int64(5), total)

	// Test second page
	filters.Page = 2
	bets, total, err = suite.repo.GetBetsByUser(ctx, user.ID, filters)
	suite.AssertNoDBError(err)
	suite.Assert().Len(bets, 2)
	suite.Assert().Equal(int64(5), total)

	// Test third page
	filters.Page = 3
	bets, total, err = suite.repo.GetBetsByUser(ctx, user.ID, filters)
	suite.AssertNoDBError(err)
	suite.Assert().Len(bets, 1)
	suite.Assert().Equal(int64(5), total)
}

func (suite *PredictionRepositoryTestSuite) TestGetBetsByUser_InvalidPagination() {
	ctx := context.Background()
	user := suite.createTestUser()
	suite.createTestBetForUser(user.ID)

	// Test invalid page (should default to 1)
	filters := &BetFilters{Page: 0, PerPage: 10}
	bets, _, err := suite.repo.GetBetsByUser(ctx, user.ID, filters)
	suite.AssertNoDBError(err)
	suite.Assert().Len(bets, 1)

	// Test invalid per_page (should default to 20)
	filters = &BetFilters{Page: 1, PerPage: 0}
	bets, _, err = suite.repo.GetBetsByUser(ctx, user.ID, filters)
	suite.AssertNoDBError(err)
	suite.Assert().Len(bets, 1)

	// Test per_page > 100 (should limit to 100)
	filters = &BetFilters{Page: 1, PerPage: 150}
	bets, _, err = suite.repo.GetBetsByUser(ctx, user.ID, filters)
	suite.AssertNoDBError(err)
	suite.Assert().Len(bets, 1)
}

func (suite *PredictionRepositoryTestSuite) TestGetBetsByMarket() {
	ctx := context.Background()
	market := suite.createTestMarket()

	// Create active bets
	bet1 := suite.createTestBetForMarket(market.ID, models.BetStatusActive)
	bet2 := suite.createTestBetForMarket(market.ID, models.BetStatusActive)

	// Create settled bet (should not be returned by this specific repo function version)
	// Note: The original GetBetsByMarket only fetches active bets.
	suite.createTestBetForMarket(market.ID, models.BetStatusSettled)

	bets, err := suite.repo.GetBetsByMarket(ctx, market.ID)
	suite.AssertNoDBError(err)
	// This will still be 2 because GetBetsByMarket filters for active status internally.
	suite.Assert().Len(bets, 2)

	betIDs := []uuid.UUID{bets[0].ID, bets[1].ID}
	suite.Assert().Contains(betIDs, bet1.ID)
	suite.Assert().Contains(betIDs, bet2.ID)
}

func (suite *PredictionRepositoryTestSuite) TestGetActiveBetsByUser() {
	ctx := context.Background()
	user := suite.createTestUser()

	// Create active bets
	bet1 := suite.createTestBetForUserWithStatus(user.ID, models.BetStatusActive)
	bet2 := suite.createTestBetForUserWithStatus(user.ID, models.BetStatusActive)

	// Create settled bet (should not be returned)
	suite.createTestBetForUserWithStatus(user.ID, models.BetStatusSettled)

	bets, err := suite.repo.GetActiveBetsByUser(ctx, user.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Len(bets, 2)

	betIDs := []uuid.UUID{bets[0].ID, bets[1].ID}
	suite.Assert().Contains(betIDs, bet1.ID)
	suite.Assert().Contains(betIDs, bet2.ID)
}

func (suite *PredictionRepositoryTestSuite) TestCreateBet() {
	ctx := context.Background()
	user := suite.createTestUser()
	market := suite.createTestMarket()
	outcome := suite.createTestOutcome(market.ID)       // Uses updated createTestOutcome
	transaction := suite.createTestTransaction(user.ID) // Uses updated createTestWallet via FirstOrCreate

	bet := &models.Bet{
		UserID:           user.ID,
		MarketID:         market.ID,
		MarketOutcomeID:  outcome.ID,
		Amount:           decimal.NewFromFloat(100),
		ContractsBought:  decimal.NewFromFloat(10),
		PricePerContract: decimal.NewFromFloat(0.5),
		TotalCost:        decimal.NewFromFloat(50),
		TransactionID:    transaction.ID,
		Status:           models.BetStatusActive,
	}

	err := suite.repo.CreateBet(ctx, bet)
	suite.AssertNoDBError(err)
	suite.Assert().NotEqual(uuid.Nil, bet.ID)

	// Verify bet was created
	count := suite.CountRecords("bets")
	suite.Assert().Equal(int64(1), count)
}

func (suite *PredictionRepositoryTestSuite) TestUpdateBet() {
	ctx := context.Background()
	bet := suite.createTestBet()

	// Update bet status
	bet.Status = models.BetStatusSettled
	now := time.Now()
	bet.SettledAt = &now
	amount := decimal.NewFromFloat(150)
	bet.SettlementAmount = &amount

	err := suite.repo.UpdateBet(ctx, bet)
	suite.AssertNoDBError(err)

	// Verify update
	updated, err := suite.repo.GetBetByID(ctx, bet.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Equal(models.BetStatusSettled, updated.Status)
	suite.Assert().NotNil(updated.SettledAt)
	suite.Assert().NotNil(updated.SettlementAmount)
}

func (suite *PredictionRepositoryTestSuite) TestGetUserPositionInMarket() {
	ctx := context.Background()
	user := suite.createTestUser()
	market := suite.createTestMarket()

	// Create active bets
	suite.createTestBetWithAmount(user.ID, market.ID, decimal.NewFromFloat(100))
	suite.createTestBetWithAmount(user.ID, market.ID, decimal.NewFromFloat(200))

	// Create settled bet (should not be counted)
	bet := suite.createTestBetWithAmount(user.ID, market.ID, decimal.NewFromFloat(50))
	bet.Status = models.BetStatusSettled
	err := suite.repo.UpdateBet(ctx, bet) // ensure this is saved to db
	suite.AssertNoDBError(err)

	position, err := suite.repo.GetUserPositionInMarket(ctx, user.ID, market.ID)
	suite.AssertNoDBError(err)
	expected := decimal.NewFromFloat(300) // 100 + 200
	suite.Assert().True(expected.Equal(position))
}

func (suite *PredictionRepositoryTestSuite) TestGetUserPositionInMarket_NoBets() {
	ctx := context.Background()
	user := suite.createTestUser()
	market := suite.createTestMarket()

	position, err := suite.repo.GetUserPositionInMarket(ctx, user.ID, market.ID)
	suite.AssertNoDBError(err)
	suite.Assert().True(decimal.Zero.Equal(position))
}

func (suite *PredictionRepositoryTestSuite) TestGetUserDailyBetAmount() {
	ctx := context.Background()
	user := suite.createTestUser()

	testDate := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	// Create bets for the test date
	bet1 := suite.createTestBetForUser(user.ID)
	bet1.CreatedAt = testDate
	bet1.Amount = decimal.NewFromFloat(100)
	err := suite.DB.Save(bet1).Error // Save changes to CreatedAt and Amount
	suite.AssertNoDBError(err)

	bet2 := suite.createTestBetForUser(user.ID)
	bet2.CreatedAt = testDate.Add(5 * time.Hour)
	bet2.Amount = decimal.NewFromFloat(200)
	err = suite.DB.Save(bet2).Error
	suite.AssertNoDBError(err)

	// Create bet for different date (should not be counted)
	bet3 := suite.createTestBetForUser(user.ID)
	bet3.CreatedAt = testDate.Add(25 * time.Hour) // Next day
	bet3.Amount = decimal.NewFromFloat(50)
	err = suite.DB.Save(bet3).Error
	suite.AssertNoDBError(err)

	dailyAmount, err := suite.repo.GetUserDailyBetAmount(ctx, user.ID, testDate)
	suite.AssertNoDBError(err)
	expected := decimal.NewFromFloat(300) // 100 + 200
	suite.Assert().True(expected.Equal(dailyAmount))
}

func (suite *PredictionRepositoryTestSuite) TestGetUserBetCount() {
	ctx := context.Background()
	user := suite.createTestUser()

	since := time.Now().Add(-24 * time.Hour)

	// Create bets after the since time
	bet1 := suite.createTestBetForUser(user.ID)
	bet1.CreatedAt = since.Add(time.Hour)
	err := suite.DB.Save(bet1).Error
	suite.AssertNoDBError(err)

	bet2 := suite.createTestBetForUser(user.ID)
	bet2.CreatedAt = since.Add(2 * time.Hour)
	err = suite.DB.Save(bet2).Error
	suite.AssertNoDBError(err)

	// Create bet before since time (should not be counted)
	bet3 := suite.createTestBetForUser(user.ID)
	bet3.CreatedAt = since.Add(-time.Hour)
	err = suite.DB.Save(bet3).Error
	suite.AssertNoDBError(err)

	count, err := suite.repo.GetUserBetCount(ctx, user.ID, since)
	suite.AssertNoDBError(err)
	suite.Assert().Equal(2, count)
}

func (suite *PredictionRepositoryTestSuite) TestGetMarketWithOutcomes() {
	ctx := context.Background()
	market := suite.createTestMarket()
	outcome1 := suite.createTestOutcome(market.ID)
	outcome2 := suite.createTestOutcome(market.ID) // Second call creates a unique outcome key

	result, err := suite.repo.GetMarketWithOutcomes(ctx, market.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Equal(market.ID, result.ID)
	suite.Assert().Len(result.Outcomes, 2)

	outcomeIDs := []uuid.UUID{result.Outcomes[0].ID, result.Outcomes[1].ID}
	suite.Assert().Contains(outcomeIDs, outcome1.ID)
	suite.Assert().Contains(outcomeIDs, outcome2.ID)
}

func (suite *PredictionRepositoryTestSuite) TestGetMarketOutcome() {
	ctx := context.Background()
	market := suite.createTestMarket()
	outcome := suite.createTestOutcome(market.ID)

	result, err := suite.repo.GetMarketOutcome(ctx, outcome.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Equal(outcome.ID, result.ID)
	suite.Assert().NotNil(result.Market)
	suite.Assert().Equal(market.ID, result.Market.ID)
}

func (suite *PredictionRepositoryTestSuite) TestUpdateMarketOutcome() {
	ctx := context.Background()
	market := suite.createTestMarket()
	outcome := suite.createTestOutcome(market.ID)

	// Update outcome
	outcome.PoolAmount = decimal.NewFromFloat(500)
	outcome.SetAsWinner()

	err := suite.repo.UpdateMarketOutcome(ctx, outcome)
	suite.AssertNoDBError(err)

	// Verify update
	updated, err := suite.repo.GetMarketOutcome(ctx, outcome.ID)
	suite.AssertNoDBError(err)
	suite.Assert().True(decimal.NewFromFloat(500).Equal(updated.PoolAmount))
	suite.Assert().True(updated.IsWinner())
}

func (suite *PredictionRepositoryTestSuite) TestGetUserWallet() {
	ctx := context.Background()
	user := suite.createTestUser()
	// Use the getOrCreate logic now embedded in createTestWallet
	wallet := suite.createTestWallet(user.ID, "NGN") // Use a different currency to ensure it's created fresh for this test if needed

	result, err := suite.repo.GetUserWallet(ctx, user.ID, "NGN")
	suite.AssertNoDBError(err)
	suite.Assert().Equal(wallet.ID, result.ID)
	suite.Assert().Equal("NGN", result.CurrencyCode)
}

func (suite *PredictionRepositoryTestSuite) TestGetUserWallet_NotFound() {
	ctx := context.Background()
	user := suite.createTestUser()

	// Attempt to get a wallet for a currency that hasn't been created for this user
	result, err := suite.repo.GetUserWallet(ctx, user.ID, "EUR")
	suite.AssertDBError(err)
	suite.Assert().Nil(result)
	suite.Assert().ErrorIs(err, gorm.ErrRecordNotFound)
}

func (suite *PredictionRepositoryTestSuite) TestCreateTransaction() {
	ctx := context.Background()
	user := suite.createTestUser()
	wallet := suite.createTestWallet(user.ID, "USD") // This will get or create

	transaction := &models.Transaction{
		UserID:          user.ID,
		WalletID:        wallet.ID,
		TransactionType: models.TransactionTypeBetPlace,
		Amount:          decimal.NewFromFloat(-100),
		BalanceBefore:   decimal.NewFromFloat(1000),
		BalanceAfter:    decimal.NewFromFloat(900),
		Description:     "Test bet placement",
	}

	err := suite.repo.CreateTransaction(ctx, transaction)
	suite.AssertNoDBError(err)
	suite.Assert().NotEqual(uuid.Nil, transaction.ID)
}

func (suite *PredictionRepositoryTestSuite) TestApplyBetSorting_SQLInjectionProtection() {
	ctx := context.Background()
	user := suite.createTestUser()
	suite.createTestBetForUser(user.ID)

	// Test with invalid sort field (should default to created_at)
	filters := &BetFilters{
		SortBy:    "invalid_field; DROP TABLE bets; --",
		SortOrder: "desc",
		Page:      1,
		PerPage:   10,
	}

	bets, _, err := suite.repo.GetBetsByUser(ctx, user.ID, filters)
	suite.AssertNoDBError(err) // Should not error due to validation
	suite.Assert().Len(bets, 1)

	// Test with invalid sort order (should default to desc)
	filters.SortBy = "amount"
	filters.SortOrder = "invalid; DROP TABLE bets; --"

	bets, _, err = suite.repo.GetBetsByUser(ctx, user.ID, filters)
	suite.AssertNoDBError(err)
	suite.Assert().Len(bets, 1)
}

// Helper methods
func (suite *PredictionRepositoryTestSuite) createTestUser() *models.User {
	country := suite.createTestCountry()
	isActive := true
	id := uuid.New()
	user := &models.User{
		CountryID:    country.ID,
		Email:        "test" + id.String()[:8] + "@example.com",
		PasswordHash: "hashedpassword",
		IsActive:     &isActive,
	}
	err := suite.DB.Create(user).Error
	suite.AssertNoDBError(err)
	return user
}

func (suite *PredictionRepositoryTestSuite) createTestCountry() *models.Country {
	isActive := true
	id := uuid.New()
	country := &models.Country{
		Name:           "Test Country " + id.String()[:8],
		Code:           "T" + id.String()[:2], // Max 3 chars
		CurrencyCode:   "USD",                 // Default currency for tests, can be overridden if specific test needs it
		CurrencySymbol: "$",
		IsActive:       &isActive,
		Config: &models.CountryConfig{
			MinBet: decimal.NewFromFloat(1),
			MaxBet: decimal.NewFromFloat(1000),
		},
	}
	// Use FirstOrCreate for countries as well to avoid test collisions if names/codes are repeated
	err := suite.DB.Where(models.Country{Code: country.Code}).FirstOrCreate(country).Error
	suite.AssertNoDBError(err)
	return country
}

func (suite *PredictionRepositoryTestSuite) createTestMarket() *models.Market {
	country := suite.createTestCountry()
	category := suite.createTestCategory(country.ID)
	id := uuid.New()
	market := &models.Market{
		CountryID:           country.ID,
		CategoryID:          category.ID,
		Title:               "Test Market " + id.String()[:8],
		Description:         "Test Description",
		MarketType:          models.MarketTypeBinary,
		Status:              models.MarketStatusOpen,
		CloseTime:           time.Now().Add(24 * time.Hour),
		ResolutionDeadline:  time.Now().Add(48 * time.Hour),
		MinBetAmount:        decimal.NewFromFloat(10),
		TotalPoolAmount:     decimal.Zero,
		RakePercentage:      decimal.NewFromFloat(0.05),
		CreatorRevenueShare: decimal.NewFromFloat(0.5),
	}
	err := suite.DB.Create(market).Error
	suite.AssertNoDBError(err)
	return market
}

func (suite *PredictionRepositoryTestSuite) createTestCategory(countryID uuid.UUID) *models.Category {
	id := uuid.New()
	category := &models.Category{
		CountryID:   countryID,
		Name:        "Test Category " + id.String()[:8],
		Slug:        "test-category-" + id.String()[:8],
		Description: "Test Description",
		IsActive:    true,
	}
	// Use FirstOrCreate for categories to avoid test collisions if slugs are repeated per country
	err := suite.DB.Where(models.Category{CountryID: countryID, Slug: category.Slug}).FirstOrCreate(category).Error
	suite.AssertNoDBError(err)
	return category
}

// Modified createTestOutcome
func (suite *PredictionRepositoryTestSuite) createTestOutcome(marketID uuid.UUID) *models.MarketOutcome {
	uniqueSuffix := uuid.New().String()[:6]
	outcome := &models.MarketOutcome{
		MarketID:     marketID,
		OutcomeKey:   "outcome-" + uniqueSuffix, // Ensure unique key
		OutcomeLabel: "Outcome " + uniqueSuffix,
		PoolAmount:   decimal.Zero,
	}
	err := suite.DB.Create(outcome).Error
	suite.AssertNoDBError(err)
	return outcome
}

// Modified createTestWallet to use FirstOrCreate
func (suite *PredictionRepositoryTestSuite) createTestWallet(userID uuid.UUID, currency string) *models.Wallet {
	wallet := &models.Wallet{
		UserID:       userID,
		CurrencyCode: currency,
	}
	// GORM's FirstOrCreate will find by UserID and CurrencyCode or create if not exists.
	// Attrs provides values only if the record is created.
	err := suite.DB.Where(models.Wallet{UserID: userID, CurrencyCode: currency}).
		Attrs(models.Wallet{Balance: decimal.NewFromFloat(1000), LockedBalance: decimal.Zero}).
		FirstOrCreate(wallet).Error

	suite.AssertNoDBError(err)
	return wallet
}

func (suite *PredictionRepositoryTestSuite) createTestTransaction(userID uuid.UUID) *models.Transaction {
	wallet := suite.createTestWallet(userID, "USD") // This will now get or create the USD wallet
	transaction := &models.Transaction{
		UserID:          userID,
		WalletID:        wallet.ID,
		TransactionType: models.TransactionTypeBetPlace,
		Amount:          decimal.NewFromFloat(-100),
		BalanceBefore:   wallet.Balance,                                // Use actual balance before
		BalanceAfter:    wallet.Balance.Sub(decimal.NewFromFloat(100)), // Calculate based on actual
		Description:     "Test transaction",
	}
	// Simulate balance update if transaction implies it (though wallet methods should handle this in real code)
	// For test helper, ensure the transaction itself is consistent.
	// The wallet balance itself isn't directly updated by this helper, only the transaction record.
	err := suite.DB.Create(transaction).Error
	suite.AssertNoDBError(err)
	return transaction
}

func (suite *PredictionRepositoryTestSuite) createTestBet() *models.Bet {
	user := suite.createTestUser()
	market := suite.createTestMarket()
	outcome := suite.createTestOutcome(market.ID)
	transaction := suite.createTestTransaction(user.ID)

	bet := &models.Bet{
		UserID:           user.ID,
		MarketID:         market.ID,
		MarketOutcomeID:  outcome.ID,
		Amount:           decimal.NewFromFloat(100),
		ContractsBought:  decimal.NewFromFloat(10),
		PricePerContract: decimal.NewFromFloat(0.5), // Should align with Amount/ContractsBought if PricePerContract is price/100
		TotalCost:        decimal.NewFromFloat(50),  // This might need to be Amount if no fees explicitly handled
		TransactionID:    transaction.ID,
		Status:           models.BetStatusActive,
	}
	// Adjust TotalCost if it's always equal to Amount for these tests
	bet.TotalCost = bet.Amount

	err := suite.DB.Create(bet).Error
	suite.AssertNoDBError(err)
	return bet
}

func (suite *PredictionRepositoryTestSuite) createTestBetForUser(userID uuid.UUID) *models.Bet {
	market := suite.createTestMarket() // Creates a new market each time
	outcome := suite.createTestOutcome(market.ID)
	transaction := suite.createTestTransaction(userID)

	bet := &models.Bet{
		UserID:           userID,
		MarketID:         market.ID,
		MarketOutcomeID:  outcome.ID,
		Amount:           decimal.NewFromFloat(100),
		ContractsBought:  decimal.NewFromFloat(10),
		PricePerContract: decimal.NewFromFloat(0.5),
		TotalCost:        decimal.NewFromFloat(100), // Assuming TotalCost is same as Amount for simplicity
		TransactionID:    transaction.ID,
		Status:           models.BetStatusActive,
	}
	err := suite.DB.Create(bet).Error
	suite.AssertNoDBError(err)
	return bet
}

func (suite *PredictionRepositoryTestSuite) createTestBetWithAmount(userID, marketID uuid.UUID, amount decimal.Decimal) *models.Bet {
	outcome := suite.createTestOutcome(marketID)
	transaction := suite.createTestTransaction(userID)

	bet := &models.Bet{
		UserID:           userID,
		MarketID:         marketID,
		MarketOutcomeID:  outcome.ID,
		Amount:           amount,
		ContractsBought:  decimal.NewFromFloat(10), // Keep other values constant or derive them
		PricePerContract: decimal.NewFromFloat(0.5),
		TotalCost:        amount, // Assuming TotalCost is same as Amount
		TransactionID:    transaction.ID,
		Status:           models.BetStatusActive,
	}
	err := suite.DB.Create(bet).Error
	suite.AssertNoDBError(err)
	return bet
}

func (suite *PredictionRepositoryTestSuite) createTestBetForMarket(marketID uuid.UUID, status models.BetStatus) *models.Bet {
	user := suite.createTestUser()
	outcome := suite.createTestOutcome(marketID) // This will create a unique outcome for this market
	transaction := suite.createTestTransaction(user.ID)

	bet := &models.Bet{
		UserID:           user.ID,
		MarketID:         marketID,
		MarketOutcomeID:  outcome.ID,
		Amount:           decimal.NewFromFloat(100),
		ContractsBought:  decimal.NewFromFloat(10),
		PricePerContract: decimal.NewFromFloat(0.5),
		TotalCost:        decimal.NewFromFloat(100),
		TransactionID:    transaction.ID,
		Status:           status,
	}
	err := suite.DB.Create(bet).Error
	suite.AssertNoDBError(err)
	return bet
}

func (suite *PredictionRepositoryTestSuite) createTestBetForUserWithStatus(userID uuid.UUID, status models.BetStatus) *models.Bet {
	market := suite.createTestMarket()
	outcome := suite.createTestOutcome(market.ID)
	transaction := suite.createTestTransaction(userID)

	bet := &models.Bet{
		UserID:           userID,
		MarketID:         market.ID,
		MarketOutcomeID:  outcome.ID,
		Amount:           decimal.NewFromFloat(100),
		ContractsBought:  decimal.NewFromFloat(10),
		PricePerContract: decimal.NewFromFloat(0.5),
		TotalCost:        decimal.NewFromFloat(100),
		TransactionID:    transaction.ID,
		Status:           status,
	}
	err := suite.DB.Create(bet).Error
	suite.AssertNoDBError(err)
	return bet
}
