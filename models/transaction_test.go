package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestTransactionMetadata(t *testing.T) {
	t.Run("Value and Scan", func(t *testing.T) {
		metadata := TransactionMetadata{
			IPAddress:       "192.168.1.1",
			UserAgent:       "test-browser",
			PaymentProvider: "paystack",
			ExchangeRate:    decimal.NewFromFloat(1.5),
			FeeAmount:       decimal.NewFromFloat(10),
			Notes:           "Test transaction",
		}

		value, err := metadata.Value()
		assert.NoError(t, err)

		var result TransactionMetadata
		err = result.Scan(value)
		assert.NoError(t, err)
		assert.Equal(t, metadata.IPAddress, result.IPAddress)
		assert.Equal(t, metadata.PaymentProvider, result.PaymentProvider)
		assert.True(t, metadata.ExchangeRate.Equal(result.ExchangeRate))

		jsonString := string(value.([]byte))
		err = result.Scan(jsonString)
		assert.NoError(t, err)

		err = result.Scan(nil)
		assert.NoError(t, err)

		err = result.Scan(123)
		assert.NoError(t, err)
	})
}

func TestTransaction(t *testing.T) {
	t.Run("TableName", func(t *testing.T) {
		tx := Transaction{}
		assert.Equal(t, "transactions", tx.TableName())
	})

	t.Run("BeforeCreate", func(t *testing.T) {
		tx := Transaction{}
		assert.Equal(t, uuid.Nil, tx.ID)

		err := tx.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, tx.ID)

		existingID := uuid.New()
		tx2 := Transaction{ID: existingID}
		err = tx2.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, existingID, tx2.ID)
	})

	t.Run("Credit and Debit checks", func(t *testing.T) {
		credit := Transaction{Amount: decimal.NewFromFloat(100)}
		assert.True(t, credit.IsCredit())
		assert.False(t, credit.IsDebit())

		debit := Transaction{Amount: decimal.NewFromFloat(-50)}
		assert.False(t, debit.IsCredit())
		assert.True(t, debit.IsDebit())

		zero := Transaction{Amount: decimal.Zero}
		assert.False(t, zero.IsCredit())
		assert.False(t, zero.IsDebit())
	})

	t.Run("GetAbsoluteAmount", func(t *testing.T) {
		tx := Transaction{Amount: decimal.NewFromFloat(-100)}
		abs := tx.GetAbsoluteAmount()
		expected := decimal.NewFromFloat(100)
		assert.True(t, expected.Equal(abs))

		tx.Amount = decimal.NewFromFloat(50)
		abs = tx.GetAbsoluteAmount()
		expected = decimal.NewFromFloat(50)
		assert.True(t, expected.Equal(abs))
	})

	t.Run("IsBalanceConsistent", func(t *testing.T) {
		tx := Transaction{
			Amount:        decimal.NewFromFloat(100),
			BalanceBefore: decimal.NewFromFloat(200),
			BalanceAfter:  decimal.NewFromFloat(300),
		}
		assert.True(t, tx.IsBalanceConsistent())

		tx.BalanceAfter = decimal.NewFromFloat(250)
		assert.False(t, tx.IsBalanceConsistent())
	})

	t.Run("Validate", func(t *testing.T) {
		validTx := Transaction{
			UserID:        uuid.New(),
			WalletID:      uuid.New(),
			Amount:        decimal.NewFromFloat(100),
			BalanceBefore: decimal.NewFromFloat(200),
			BalanceAfter:  decimal.NewFromFloat(300),
		}
		assert.NoError(t, validTx.Validate())

		tests := []struct {
			name   string
			modify func(*Transaction)
			err    error
		}{
			{"Valid Transaction", func(_ *Transaction) {}, nil},
			{"Invalid UserID", func(tx *Transaction) { tx.UserID = uuid.Nil }, ErrInvalidUserID},
			{"Invalid WalletID", func(tx *Transaction) { tx.WalletID = uuid.Nil }, ErrInvalidWalletBalance},
			{"Zero Amount", func(tx *Transaction) { tx.Amount = decimal.Zero }, ErrInvalidTransactionAmount},
			{"Inconsistent Balance", func(tx *Transaction) { tx.BalanceAfter = decimal.NewFromFloat(250) }, ErrInvalidTransactionAmount},
			{"Negative Balance", func(tx *Transaction) {
				tx.BalanceBefore = decimal.NewFromFloat(50)
				tx.Amount = decimal.NewFromFloat(-100)
				tx.BalanceAfter = decimal.NewFromFloat(-50)
			}, ErrNegativeBalance},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				transaction := validTx
				tt.modify(&transaction)
				assert.Equal(t, tt.err, transaction.Validate())
			})
		}
	})

	t.Run("CreateDepositTransaction", func(t *testing.T) {
		userID := uuid.New()
		walletID := uuid.New()
		amount := decimal.NewFromFloat(1000)
		balanceBefore := decimal.NewFromFloat(500)
		paymentRef := uuid.New().String()

		tx := CreateDepositTransaction(userID, walletID, amount, balanceBefore, paymentRef)

		assert.Equal(t, userID, tx.UserID)
		assert.Equal(t, walletID, tx.WalletID)
		assert.Equal(t, TransactionTypeDeposit, tx.TransactionType)
		assert.True(t, amount.Equal(tx.Amount))
		assert.True(t, balanceBefore.Equal(tx.BalanceBefore))
		assert.True(t, decimal.NewFromFloat(1500).Equal(tx.BalanceAfter))
		assert.Equal(t, "payment", tx.ReferenceType)
		assert.NotNil(t, tx.ReferenceID)
	})

	t.Run("CreateWithdrawalTransaction", func(t *testing.T) {
		userID := uuid.New()
		walletID := uuid.New()
		amount := decimal.NewFromFloat(300)
		balanceBefore := decimal.NewFromFloat(1000)
		paymentRef := uuid.New().String()

		tx := CreateWithdrawalTransaction(userID, walletID, amount, balanceBefore, paymentRef)

		assert.Equal(t, TransactionTypeWithdrawal, tx.TransactionType)
		assert.True(t, amount.Neg().Equal(tx.Amount))
		assert.True(t, decimal.NewFromFloat(700).Equal(tx.BalanceAfter))
		assert.Equal(t, "payment", tx.ReferenceType)
	})

	t.Run("CreateBetTransaction", func(t *testing.T) {
		userID := uuid.New()
		walletID := uuid.New()
		amount := decimal.NewFromFloat(100)
		balanceBefore := decimal.NewFromFloat(500)
		betID := uuid.New()

		tx := CreateBetTransaction(userID, walletID, amount, balanceBefore, betID)

		assert.Equal(t, TransactionTypeBetPlace, tx.TransactionType)
		assert.True(t, amount.Neg().Equal(tx.Amount))
		assert.True(t, decimal.NewFromFloat(400).Equal(tx.BalanceAfter))
		assert.Equal(t, "bet", tx.ReferenceType)
		assert.Equal(t, betID, *tx.ReferenceID)
	})

	t.Run("CreatePayoutTransaction", func(t *testing.T) {
		userID := uuid.New()
		walletID := uuid.New()
		amount := decimal.NewFromFloat(200)
		balanceBefore := decimal.NewFromFloat(300)
		settlementID := uuid.New()

		tx := CreatePayoutTransaction(userID, walletID, amount, balanceBefore, settlementID)

		assert.Equal(t, TransactionTypePayout, tx.TransactionType)
		assert.True(t, amount.Equal(tx.Amount))
		assert.True(t, decimal.NewFromFloat(500).Equal(tx.BalanceAfter))
		assert.Equal(t, "settlement", tx.ReferenceType)
		assert.Equal(t, settlementID, *tx.ReferenceID)
	})

	t.Run("parseUUIDPtr", func(t *testing.T) {
		validUUID := uuid.New().String()
		result := parseUUIDPtr(validUUID)
		assert.NotNil(t, result)
		assert.Equal(t, validUUID, result.String())

		result = parseUUIDPtr("")
		assert.Nil(t, result)

		result = parseUUIDPtr("invalid-uuid")
		assert.Nil(t, result)
	})
}
