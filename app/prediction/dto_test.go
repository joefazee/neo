package prediction

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joefazee/neo/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestToBetResponse(t *testing.T) {
	userID := uuid.New()
	marketID := uuid.New()
	outcomeID := uuid.New()
	transactionID := uuid.New()
	now := time.Now()
	settlementAmount := decimal.NewFromFloat(150.0)

	baseBet := &models.Bet{
		ID:               uuid.New(),
		UserID:           userID,
		MarketID:         marketID,
		MarketOutcomeID:  outcomeID,
		Amount:           decimal.NewFromFloat(100.0),
		ContractsBought:  decimal.NewFromFloat(10.0),
		PricePerContract: decimal.NewFromFloat(10.0),
		TotalCost:        decimal.NewFromFloat(100.0),
		TransactionID:    transactionID,
		Status:           models.BetStatusActive,
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now,
	}

	market := &models.Market{
		ID:        marketID,
		Title:     "Test Market",
		Status:    models.MarketStatusOpen,
		CloseTime: now.Add(24 * time.Hour),
	}

	outcome := &models.MarketOutcome{
		ID:           outcomeID,
		MarketID:     marketID,
		OutcomeKey:   "yes",
		OutcomeLabel: "Yes",
	}

	t.Run("Full Bet Data", func(t *testing.T) {
		bet := *baseBet
		bet.Market = market
		bet.MarketOutcome = outcome
		bet.SettledAt = &now
		bet.SettlementAmount = &settlementAmount

		dto := ToBetResponse(&bet)

		assert.Equal(t, bet.ID, dto.ID)
		assert.Equal(t, bet.UserID, dto.UserID)
		assert.Equal(t, bet.MarketID, dto.MarketID)
		assert.Equal(t, bet.MarketOutcomeID, dto.OutcomeID)
		assert.True(t, bet.Amount.Equal(dto.Amount))
		assert.True(t, bet.ContractsBought.Equal(dto.ContractsBought))
		assert.True(t, bet.PricePerContract.Equal(dto.PricePerContract))
		assert.True(t, bet.TotalCost.Equal(dto.TotalCost))
		assert.Equal(t, string(bet.Status), dto.Status)
		assert.Equal(t, bet.CreatedAt, dto.PlacedAt)
		assert.NotNil(t, dto.SettledAt)
		assert.Equal(t, *bet.SettledAt, *dto.SettledAt)
		assert.NotNil(t, dto.SettlementAmount)
		assert.True(t, bet.SettlementAmount.Equal(*dto.SettlementAmount))

		assert.NotNil(t, dto.Market)
		assert.Equal(t, market.ID, dto.Market.ID)
		assert.Equal(t, market.Title, dto.Market.Title)
		assert.Equal(t, string(market.Status), dto.Market.Status)
		assert.Equal(t, market.CloseTime, dto.Market.CloseTime)

		assert.NotNil(t, dto.Outcome)
		assert.Equal(t, outcome.ID, dto.Outcome.ID)
		assert.Equal(t, outcome.OutcomeKey, dto.Outcome.Key)
		assert.Equal(t, outcome.OutcomeLabel, dto.Outcome.Label)
	})

	t.Run("Nil Market", func(t *testing.T) {
		bet := *baseBet
		bet.Market = nil
		bet.MarketOutcome = outcome

		dto := ToBetResponse(&bet)
		assert.Nil(t, dto.Market, "Market should be nil in DTO if nil in model")
		assert.NotNil(t, dto.Outcome)
	})

	t.Run("Nil MarketOutcome", func(t *testing.T) {
		bet := *baseBet
		bet.Market = market
		bet.MarketOutcome = nil

		dto := ToBetResponse(&bet)
		assert.NotNil(t, dto.Market)
		assert.Nil(t, dto.Outcome, "Outcome should be nil in DTO if nil in model")
	})

	t.Run("Nil Market and MarketOutcome", func(t *testing.T) {
		bet := *baseBet
		bet.Market = nil
		bet.MarketOutcome = nil

		dto := ToBetResponse(&bet)
		assert.Nil(t, dto.Market)
		assert.Nil(t, dto.Outcome)
	})

	t.Run("Nil SettledAt and SettlementAmount", func(t *testing.T) {
		bet := *baseBet
		bet.Market = market
		bet.MarketOutcome = outcome
		bet.SettledAt = nil
		bet.SettlementAmount = nil

		dto := ToBetResponse(&bet)
		assert.Nil(t, dto.SettledAt)
		assert.Nil(t, dto.SettlementAmount)
	})
}

func TestToBetResponseList(t *testing.T) {
	userID := uuid.New()
	marketID1 := uuid.New()
	outcomeID1 := uuid.New()
	marketID2 := uuid.New()
	outcomeID2 := uuid.New()
	now := time.Now()

	bet1 := models.Bet{
		ID:              uuid.New(),
		UserID:          userID,
		MarketID:        marketID1,
		MarketOutcomeID: outcomeID1,
		Amount:          decimal.NewFromInt(100),
		Status:          models.BetStatusActive,
		CreatedAt:       now.Add(-2 * time.Hour),
		Market: &models.Market{
			ID:    marketID1,
			Title: "Market 1",
		},
		MarketOutcome: &models.MarketOutcome{
			ID:           outcomeID1,
			OutcomeKey:   "key1",
			OutcomeLabel: "Label 1",
		},
	}

	bet2 := models.Bet{
		ID:              uuid.New(),
		UserID:          userID,
		MarketID:        marketID2,
		MarketOutcomeID: outcomeID2,
		Amount:          decimal.NewFromInt(200),
		Status:          models.BetStatusSettled,
		CreatedAt:       now.Add(-1 * time.Hour),
		SettledAt:       &now,
	}

	t.Run("Empty List", func(t *testing.T) {
		bets := []models.Bet{}
		dtos := ToBetResponseList(bets)
		assert.Empty(t, dtos)
		assert.Len(t, dtos, 0)
	})

	t.Run("List with Multiple Bets", func(t *testing.T) {
		bets := []models.Bet{bet1, bet2}
		dtos := ToBetResponseList(bets)

		assert.Len(t, dtos, 2)

		assert.Equal(t, bet1.ID, dtos[0].ID)
		assert.True(t, bet1.Amount.Equal(dtos[0].Amount))
		assert.NotNil(t, dtos[0].Market)
		assert.Equal(t, "Market 1", dtos[0].Market.Title)
		assert.NotNil(t, dtos[0].Outcome)
		assert.Equal(t, "key1", dtos[0].Outcome.Key)

		assert.Equal(t, bet2.ID, dtos[1].ID)
		assert.True(t, bet2.Amount.Equal(dtos[1].Amount))
		assert.Nil(t, dtos[1].Market)  // Was nil in model
		assert.Nil(t, dtos[1].Outcome) // Was nil in model
		assert.NotNil(t, dtos[1].SettledAt)
	})
}
