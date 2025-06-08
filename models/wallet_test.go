package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestWallet(t *testing.T) {
	t.Run("TableName", func(t *testing.T) {
		w := Wallet{}
		assert.Equal(t, "wallets", w.TableName())
	})

	t.Run("BeforeCreate", func(t *testing.T) {
		w := Wallet{}
		assert.Equal(t, uuid.Nil, w.ID)

		err := w.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, w.ID)

		existingID := uuid.New()
		w2 := Wallet{ID: existingID}
		err = w2.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, existingID, w2.ID)
	})

	t.Run("GetAvailableBalance", func(t *testing.T) {
		w := Wallet{
			Balance:       decimal.NewFromFloat(1000),
			LockedBalance: decimal.NewFromFloat(200),
		}

		available := w.GetAvailableBalance()
		expected := decimal.NewFromFloat(800)
		assert.True(t, expected.Equal(available))
	})

	t.Run("CanDebit", func(t *testing.T) {
		w := Wallet{
			Balance:       decimal.NewFromFloat(1000),
			LockedBalance: decimal.NewFromFloat(200),
		}

		assert.True(t, w.CanDebit(decimal.NewFromFloat(500)))
		assert.True(t, w.CanDebit(decimal.NewFromFloat(800)))
		assert.False(t, w.CanDebit(decimal.NewFromFloat(900)))
	})

	t.Run("LockFunds", func(t *testing.T) {
		w := Wallet{
			Balance:       decimal.NewFromFloat(1000),
			LockedBalance: decimal.NewFromFloat(200),
		}

		err := w.LockFunds(decimal.NewFromFloat(300))
		assert.NoError(t, err)
		expected := decimal.NewFromFloat(500)
		assert.True(t, expected.Equal(w.LockedBalance))

		err = w.LockFunds(decimal.NewFromFloat(600))
		assert.Equal(t, ErrInsufficientBalance, err)
	})

	t.Run("UnlockFunds", func(t *testing.T) {
		w := Wallet{
			Balance:       decimal.NewFromFloat(1000),
			LockedBalance: decimal.NewFromFloat(500),
		}

		err := w.UnlockFunds(decimal.NewFromFloat(200))
		assert.NoError(t, err)
		expected := decimal.NewFromFloat(300)
		assert.True(t, expected.Equal(w.LockedBalance))

		err = w.UnlockFunds(decimal.NewFromFloat(400))
		assert.Equal(t, ErrInvalidWalletBalance, err)
	})

	t.Run("Credit", func(t *testing.T) {
		w := Wallet{Balance: decimal.NewFromFloat(500)}

		err := w.Credit(decimal.NewFromFloat(200))
		assert.NoError(t, err)
		expected := decimal.NewFromFloat(700)
		assert.True(t, expected.Equal(w.Balance))

		err = w.Credit(decimal.Zero)
		assert.Equal(t, ErrInvalidTransactionAmount, err)

		err = w.Credit(decimal.NewFromFloat(-100))
		assert.Equal(t, ErrInvalidTransactionAmount, err)
	})

	t.Run("Debit", func(t *testing.T) {
		w := Wallet{
			Balance:       decimal.NewFromFloat(1000),
			LockedBalance: decimal.NewFromFloat(200),
		}

		err := w.Debit(decimal.NewFromFloat(300))
		assert.NoError(t, err)
		expected := decimal.NewFromFloat(700)
		assert.True(t, expected.Equal(w.Balance))

		err = w.Debit(decimal.NewFromFloat(600))
		assert.Equal(t, ErrInsufficientBalance, err)

		err = w.Debit(decimal.Zero)
		assert.Equal(t, ErrInvalidTransactionAmount, err)
	})

	t.Run("DebitLocked", func(t *testing.T) {
		w := Wallet{
			Balance:       decimal.NewFromFloat(1000),
			LockedBalance: decimal.NewFromFloat(300),
		}

		err := w.DebitLocked(decimal.NewFromFloat(200))
		assert.NoError(t, err)
		assert.True(t, decimal.NewFromFloat(100).Equal(w.LockedBalance))
		assert.True(t, decimal.NewFromFloat(800).Equal(w.Balance))

		err = w.DebitLocked(decimal.NewFromFloat(200))
		assert.Equal(t, ErrInsufficientBalance, err)

		err = w.DebitLocked(decimal.Zero)
		assert.Equal(t, ErrInvalidTransactionAmount, err)
	})

	t.Run("Validate", func(t *testing.T) {
		validWallet := Wallet{
			UserID:        uuid.New(),
			CurrencyCode:  "USD",
			Balance:       decimal.NewFromFloat(1000),
			LockedBalance: decimal.NewFromFloat(200),
		}
		assert.NoError(t, validWallet.Validate())

		tests := []struct {
			name   string
			modify func(*Wallet)
			err    error
		}{
			{"Valid Wallet", func(_ *Wallet) {}, nil},
			{"Invalid UserID", func(w *Wallet) { w.UserID = uuid.Nil }, ErrInvalidUserID},
			{"Invalid Currency", func(w *Wallet) { w.CurrencyCode = "US" }, ErrInvalidCurrencyCode},
			{"Negative Balance", func(w *Wallet) { w.Balance = decimal.NewFromFloat(-10) }, ErrNegativeBalance},
			{"Negative Locked", func(w *Wallet) { w.LockedBalance = decimal.NewFromFloat(-5) }, ErrNegativeBalance},
			{"Locked > Balance", func(w *Wallet) { w.LockedBalance = decimal.NewFromFloat(1200) }, ErrInvalidWalletBalance},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				wallet := validWallet
				tt.modify(&wallet)
				if tt.err != nil {
					assert.Equal(t, tt.err, wallet.Validate())
				} else {
					assert.NoError(t, wallet.Validate())
				}
			})
		}
	})
}
