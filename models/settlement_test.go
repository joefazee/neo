package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestSettlement(t *testing.T) {
	t.Run("TableName", func(t *testing.T) {
		s := Settlement{}
		assert.Equal(t, "settlements", s.TableName())
	})

	t.Run("BeforeCreate", func(t *testing.T) {
		s := Settlement{}
		assert.Equal(t, uuid.Nil, s.ID)

		err := s.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, s.ID)

		existingID := uuid.New()
		s2 := Settlement{ID: existingID}
		err = s2.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, existingID, s2.ID)
	})

	t.Run("Settlement type checks", func(t *testing.T) {
		tests := []struct {
			settlementType SettlementType
			isWin          bool
			isLoss         bool
			isRefund       bool
		}{
			{SettlementTypeWin, true, false, false},
			{SettlementTypeLoss, false, true, false},
			{SettlementTypeRefund, false, false, true},
		}

		for _, tt := range tests {
			s := Settlement{SettlementType: tt.settlementType}
			assert.Equal(t, tt.isWin, s.IsWin())
			assert.Equal(t, tt.isLoss, s.IsLoss())
			assert.Equal(t, tt.isRefund, s.IsRefund())
		}
	})

	t.Run("GetNetAmount", func(t *testing.T) {
		s := Settlement{
			OriginalAmount: decimal.NewFromFloat(100),
			PayoutAmount:   decimal.NewFromFloat(150),
		}

		netAmount := s.GetNetAmount()
		expected := decimal.NewFromFloat(50) // 150 - 100
		assert.True(t, expected.Equal(netAmount))

		s.PayoutAmount = decimal.NewFromFloat(80)
		netAmount = s.GetNetAmount()
		expected = decimal.NewFromFloat(-20) // 80 - 100
		assert.True(t, expected.Equal(netAmount))
	})

	t.Run("GetReturnMultiple", func(t *testing.T) {
		s := Settlement{
			OriginalAmount: decimal.NewFromFloat(100),
			PayoutAmount:   decimal.NewFromFloat(150),
		}

		returnMultiple := s.GetReturnMultiple()
		expected := decimal.NewFromFloat(1.5) // 150/100
		assert.True(t, expected.Equal(returnMultiple))

		s.OriginalAmount = decimal.Zero
		returnMultiple = s.GetReturnMultiple()
		assert.True(t, decimal.Zero.Equal(returnMultiple))
	})

	t.Run("Validate", func(t *testing.T) {
		validSettlement := Settlement{
			MarketID:       uuid.New(),
			UserID:         uuid.New(),
			BetID:          uuid.New(),
			OriginalAmount: decimal.NewFromFloat(100),
			PayoutAmount:   decimal.NewFromFloat(150),
			RakeAmount:     decimal.NewFromFloat(5),
		}
		assert.NoError(t, validSettlement.Validate())

		tests := []struct {
			name   string
			modify func(*Settlement)
			err    error
		}{
			{"Invalid MarketID", func(s *Settlement) { s.MarketID = uuid.Nil }, ErrInvalidMarketID},
			{"Invalid UserID", func(s *Settlement) { s.UserID = uuid.Nil }, ErrInvalidUserID},
			{"Invalid BetID", func(s *Settlement) { s.BetID = uuid.Nil }, ErrInvalidBetAmount},
			{"Invalid OriginalAmount", func(s *Settlement) { s.OriginalAmount = decimal.Zero }, ErrInvalidBetAmount},
			{"Negative PayoutAmount", func(s *Settlement) { s.PayoutAmount = decimal.NewFromFloat(-10) }, ErrInvalidTransactionAmount},
			{"Negative RakeAmount", func(s *Settlement) { s.RakeAmount = decimal.NewFromFloat(-5) }, ErrInvalidTransactionAmount},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				settlement := validSettlement
				tt.modify(&settlement)
				assert.Equal(t, tt.err, settlement.Validate())
			})
		}
	})

	t.Run("CreateWinSettlement", func(t *testing.T) {
		marketID := uuid.New()
		userID := uuid.New()
		betID := uuid.New()
		originalAmount := decimal.NewFromFloat(100)
		payoutAmount := decimal.NewFromFloat(150)
		rakeAmount := decimal.NewFromFloat(5)

		settlement := CreateWinSettlement(marketID, userID, betID, originalAmount, payoutAmount, rakeAmount)

		assert.Equal(t, marketID, settlement.MarketID)
		assert.Equal(t, userID, settlement.UserID)
		assert.Equal(t, betID, settlement.BetID)
		assert.Equal(t, SettlementTypeWin, settlement.SettlementType)
		assert.True(t, originalAmount.Equal(settlement.OriginalAmount))
		assert.True(t, payoutAmount.Equal(settlement.PayoutAmount))
		assert.True(t, rakeAmount.Equal(settlement.RakeAmount))
	})

	t.Run("CreateLossSettlement", func(t *testing.T) {
		marketID := uuid.New()
		userID := uuid.New()
		betID := uuid.New()
		originalAmount := decimal.NewFromFloat(100)

		settlement := CreateLossSettlement(marketID, userID, betID, originalAmount)

		assert.Equal(t, marketID, settlement.MarketID)
		assert.Equal(t, userID, settlement.UserID)
		assert.Equal(t, betID, settlement.BetID)
		assert.Equal(t, SettlementTypeLoss, settlement.SettlementType)
		assert.True(t, originalAmount.Equal(settlement.OriginalAmount))
		assert.True(t, decimal.Zero.Equal(settlement.PayoutAmount))
		assert.True(t, decimal.Zero.Equal(settlement.RakeAmount))
	})

	t.Run("CreateRefundSettlement", func(t *testing.T) {
		marketID := uuid.New()
		userID := uuid.New()
		betID := uuid.New()
		originalAmount := decimal.NewFromFloat(100)

		settlement := CreateRefundSettlement(marketID, userID, betID, originalAmount)

		assert.Equal(t, marketID, settlement.MarketID)
		assert.Equal(t, userID, settlement.UserID)
		assert.Equal(t, betID, settlement.BetID)
		assert.Equal(t, SettlementTypeRefund, settlement.SettlementType)
		assert.True(t, originalAmount.Equal(settlement.OriginalAmount))
		assert.True(t, originalAmount.Equal(settlement.PayoutAmount)) // Full refund
		assert.True(t, decimal.Zero.Equal(settlement.RakeAmount))
	})
}
