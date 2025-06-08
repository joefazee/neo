package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestBetMetadata(t *testing.T) {
	t.Run("Value", func(t *testing.T) {
		metadata := BetMetadata{
			IPAddress:       "127.0.0.1",
			UserAgent:       "test-agent",
			SlippageWarning: true,
			PriceAtTime:     decimal.NewFromFloat(0.55),
		}

		value, err := metadata.Value()
		assert.NoError(t, err)
		assert.NotNil(t, value)

		var result BetMetadata
		err = json.Unmarshal(value.([]byte), &result)
		assert.NoError(t, err)
		assert.Equal(t, metadata.IPAddress, result.IPAddress)
		assert.Equal(t, metadata.UserAgent, result.UserAgent)
		assert.Equal(t, metadata.SlippageWarning, result.SlippageWarning)
		assert.True(t, metadata.PriceAtTime.Equal(result.PriceAtTime))
	})

	t.Run("Scan", func(t *testing.T) {
		jsonData := `{"ip_address":"192.168.1.1","user_agent":"browser","slippage_warning":false}`

		var metadata BetMetadata
		err := metadata.Scan([]byte(jsonData))
		assert.NoError(t, err)
		assert.Equal(t, "192.168.1.1", metadata.IPAddress)
		assert.Equal(t, "browser", metadata.UserAgent)
		assert.False(t, metadata.SlippageWarning)

		err = metadata.Scan(jsonData)
		assert.NoError(t, err)

		err = metadata.Scan(nil)
		assert.NoError(t, err)

		err = metadata.Scan(func() {})
		assert.Nil(t, err)
	})
}

func TestBet(t *testing.T) {
	t.Run("TableName", func(t *testing.T) {
		b := Bet{}
		assert.Equal(t, "bets", b.TableName())
	})

	t.Run("BeforeCreate", func(t *testing.T) {
		b := Bet{}
		assert.Equal(t, uuid.Nil, b.ID)

		err := b.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, b.ID)

		existingID := uuid.New()
		b2 := Bet{ID: existingID}
		err = b2.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, existingID, b2.ID)
	})

	t.Run("Status checks", func(t *testing.T) {
		b := Bet{Status: BetStatusActive}
		assert.True(t, b.IsActive())
		assert.False(t, b.IsSettled())
		assert.False(t, b.IsRefunded())

		now := time.Now()
		b.Status = BetStatusSettled
		b.SettledAt = &now
		assert.False(t, b.IsActive())
		assert.True(t, b.IsSettled())
		assert.False(t, b.IsRefunded())

		b.Status = BetStatusRefunded
		assert.False(t, b.IsActive())
		assert.False(t, b.IsSettled())
		assert.True(t, b.IsRefunded())
	})

	t.Run("CalculatePotentialPayout", func(t *testing.T) {
		b := Bet{ContractsBought: decimal.NewFromFloat(10)}

		totalWinning := decimal.NewFromFloat(100)
		prizePool := decimal.NewFromFloat(1000)

		payout := b.CalculatePotentialPayout(totalWinning, prizePool)
		expected := decimal.NewFromFloat(100) // (10/100) * 1000
		assert.True(t, expected.Equal(payout))

		zeroPayout := b.CalculatePotentialPayout(decimal.Zero, prizePool)
		assert.True(t, decimal.Zero.Equal(zeroPayout))
	})

	t.Run("Settle", func(t *testing.T) {
		b := Bet{Status: BetStatusActive}
		settlementAmount := decimal.NewFromFloat(150)

		err := b.Settle(settlementAmount)
		assert.NoError(t, err)
		assert.Equal(t, BetStatusSettled, b.Status)
		assert.NotNil(t, b.SettledAt)
		assert.NotNil(t, b.SettlementAmount)
		assert.True(t, settlementAmount.Equal(*b.SettlementAmount))

		err = b.Settle(decimal.NewFromFloat(200))
		assert.Equal(t, ErrBetAlreadySettled, err)
	})

	t.Run("Refund", func(t *testing.T) {
		amount := decimal.NewFromFloat(100)
		b := Bet{Status: BetStatusActive, Amount: amount}

		err := b.Refund()
		assert.NoError(t, err)
		assert.Equal(t, BetStatusRefunded, b.Status)
		assert.NotNil(t, b.SettledAt)
		assert.NotNil(t, b.SettlementAmount)
		assert.True(t, amount.Equal(*b.SettlementAmount))

		err = b.Refund()
		assert.Equal(t, ErrBetAlreadySettled, err)
	})

	t.Run("GetProfitLoss", func(t *testing.T) {
		amount := decimal.NewFromFloat(100)
		settlement := decimal.NewFromFloat(150)

		b := Bet{Status: BetStatusActive, Amount: amount}
		assert.True(t, decimal.Zero.Equal(b.GetProfitLoss()))

		b.Status = BetStatusSettled
		now := time.Now()
		b.SettledAt = &now
		assert.True(t, amount.Neg().Equal(b.GetProfitLoss()))

		b.SettlementAmount = &settlement
		profit := decimal.NewFromFloat(50) // 150 - 100
		assert.True(t, profit.Equal(b.GetProfitLoss()))
	})

	t.Run("GetReturn", func(t *testing.T) {
		amount := decimal.NewFromFloat(100)
		settlement := decimal.NewFromFloat(150)

		b := Bet{Status: BetStatusActive, Amount: amount}
		assert.True(t, decimal.Zero.Equal(b.GetReturn()))

		b.Status = BetStatusSettled
		now := time.Now()
		b.SettledAt = &now
		b.SettlementAmount = &settlement
		expected := decimal.NewFromFloat(1.5) // 150/100
		assert.True(t, expected.Equal(b.GetReturn()))

		b.Amount = decimal.Zero
		assert.True(t, decimal.Zero.Equal(b.GetReturn()))
	})

	t.Run("Validate", func(t *testing.T) {
		validBet := Bet{
			UserID:           uuid.New(),
			MarketID:         uuid.New(),
			MarketOutcomeID:  uuid.New(),
			Amount:           decimal.NewFromFloat(100),
			ContractsBought:  decimal.NewFromFloat(10),
			PricePerContract: decimal.NewFromFloat(0.5),
			TotalCost:        decimal.NewFromFloat(50),
			TransactionID:    uuid.New(),
		}
		assert.NoError(t, validBet.Validate())

		tests := []struct {
			name   string
			modify func(*Bet)
			err    error
		}{
			{"Invalid UserID", func(b *Bet) { b.UserID = uuid.Nil }, ErrInvalidUserID},
			{"Invalid MarketID", func(b *Bet) { b.MarketID = uuid.Nil }, ErrInvalidMarketID},
			{"Invalid OutcomeID", func(b *Bet) { b.MarketOutcomeID = uuid.Nil }, ErrInvalidOutcomeKey},
			{"Invalid Amount", func(b *Bet) { b.Amount = decimal.Zero }, ErrInvalidBetAmount},
			{"Invalid Contracts", func(b *Bet) { b.ContractsBought = decimal.Zero }, ErrInvalidBetAmount},
			{"Invalid Price", func(b *Bet) { b.PricePerContract = decimal.Zero }, ErrInvalidBetAmount},
			{"Invalid Cost", func(b *Bet) { b.TotalCost = decimal.Zero }, ErrInvalidBetAmount},
			{"Invalid TransactionID", func(b *Bet) { b.TransactionID = uuid.Nil }, ErrInvalidTransactionType},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				bet := validBet
				tt.modify(&bet)
				assert.Equal(t, tt.err, bet.Validate())
			})
		}
	})
}
