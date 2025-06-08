package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestProviderResponse(t *testing.T) {
	t.Run("Value and Scan", func(t *testing.T) {
		paymentDate := time.Now()
		response := ProviderResponse{
			TransactionID: "txn_123",
			Reference:     "ref_456",
			Status:        "success",
			Message:       "Payment successful",
			Gateway:       "card",
			Channel:       "visa",
			Currency:      "NGN",
			Amount:        decimal.NewFromFloat(1000),
			Fees:          decimal.NewFromFloat(15),
			Authorization: map[string]interface{}{"auth_code": "AUTH_xyz"},
			Customer:      map[string]interface{}{"email": "test@example.com"},
			PaymentDate:   &paymentDate,
			RawResponse:   map[string]interface{}{"full": "response"},
		}

		value, err := response.Value()
		assert.NoError(t, err)

		var result ProviderResponse
		err = result.Scan(value)
		assert.NoError(t, err)
		assert.Equal(t, response.TransactionID, result.TransactionID)
		assert.Equal(t, response.Reference, result.Reference)
		assert.True(t, response.Amount.Equal(result.Amount))

		jsonString := string(value.([]byte))
		err = result.Scan(jsonString)
		assert.NoError(t, err)

		err = result.Scan(nil)
		assert.NoError(t, err)

		err = result.Scan(func() {})
		assert.NoError(t, err)
	})
}

func TestPaymentTransaction(t *testing.T) {
	t.Run("TableName", func(t *testing.T) {
		pt := PaymentTransaction{}
		assert.Equal(t, "payment_transactions", pt.TableName())
	})

	t.Run("BeforeCreate", func(t *testing.T) {
		pt := PaymentTransaction{}
		assert.Equal(t, uuid.Nil, pt.ID)

		err := pt.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, pt.ID)

		existingID := uuid.New()
		pt2 := PaymentTransaction{ID: existingID}
		err = pt2.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, existingID, pt2.ID)
	})

	t.Run("Payment type checks", func(t *testing.T) {
		deposit := PaymentTransaction{PaymentType: PaymentTypeDeposit}
		assert.True(t, deposit.IsDeposit())
		assert.False(t, deposit.IsWithdrawal())

		withdrawal := PaymentTransaction{PaymentType: PaymentTypeWithdrawal}
		assert.False(t, withdrawal.IsDeposit())
		assert.True(t, withdrawal.IsWithdrawal())
	})

	t.Run("Status checks", func(t *testing.T) {
		tests := []struct {
			status     PaymentStatus
			pending    bool
			processing bool
			successful bool
			failed     bool
			canceled   bool
			completed  bool
		}{
			{PaymentStatusPending, true, false, false, false, false, false},
			{PaymentStatusProcessing, false, true, false, false, false, false},
			{PaymentStatusSuccess, false, false, true, false, false, true},
			{PaymentStatusFailed, false, false, false, true, false, true},
			{PaymentStatusCancelled, false, false, false, false, true, true},
		}

		for _, tt := range tests {
			pt := PaymentTransaction{Status: tt.status}
			assert.Equal(t, tt.pending, pt.IsPending())
			assert.Equal(t, tt.processing, pt.IsProcessing())
			assert.Equal(t, tt.successful, pt.IsSuccessful())
			assert.Equal(t, tt.failed, pt.IsFailed())
			assert.Equal(t, tt.canceled, pt.IsCancelled())
			assert.Equal(t, tt.completed, pt.IsCompleted())
		}
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		pt := PaymentTransaction{}
		response := ProviderResponse{
			TransactionID: "txn_123",
			Status:        "success",
			Amount:        decimal.NewFromFloat(1000),
		}

		pt.UpdateStatus(PaymentStatusSuccess, &response)
		assert.Equal(t, PaymentStatusSuccess, pt.Status)
		assert.Equal(t, response.TransactionID, pt.ProviderResponse.TransactionID)
	})

	t.Run("MarkWebhookVerified", func(t *testing.T) {
		pt := PaymentTransaction{WebhookVerified: false}
		pt.MarkWebhookVerified()
		assert.True(t, pt.WebhookVerified)
	})

	t.Run("Financial calculations", func(t *testing.T) {
		pt := PaymentTransaction{
			Amount: decimal.NewFromFloat(1000),
			ProviderResponse: ProviderResponse{
				Fees: decimal.NewFromFloat(15),
			},
		}

		fees := pt.GetProviderFees()
		expected := decimal.NewFromFloat(15)
		assert.True(t, expected.Equal(fees))

		netAmount := pt.GetNetAmount()
		expected = decimal.NewFromFloat(985)
		assert.True(t, expected.Equal(netAmount))
	})

	t.Run("Validate", func(t *testing.T) {
		validPayment := PaymentTransaction{
			UserID:            uuid.New(),
			Provider:          PaymentProviderPaystack,
			ProviderReference: "ref_123",
			Amount:            decimal.NewFromFloat(1000),
			CurrencyCode:      "NGN",
		}
		assert.NoError(t, validPayment.Validate())

		tests := []struct {
			name   string
			modify func(*PaymentTransaction)
			err    error
		}{
			{"Invalid UserID", func(pt *PaymentTransaction) { pt.UserID = uuid.Nil }, ErrInvalidUserID},
			{"Empty Provider", func(pt *PaymentTransaction) { pt.Provider = "" }, ErrInvalidPaymentProvider},
			{"Empty Reference", func(pt *PaymentTransaction) { pt.ProviderReference = "" }, ErrInvalidProviderReference},
			{"Invalid Amount", func(pt *PaymentTransaction) { pt.Amount = decimal.Zero }, ErrInvalidTransactionAmount},
			{"Invalid Currency", func(pt *PaymentTransaction) { pt.CurrencyCode = "NG" }, ErrInvalidCurrencyCode},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				payment := validPayment
				tt.modify(&payment)
				assert.Equal(t, tt.err, payment.Validate())
			})
		}
	})

	t.Run("CreateDepositPayment", func(t *testing.T) {
		userID := uuid.New()
		amount := decimal.NewFromFloat(1000)

		deposit := CreateDepositPayment(userID, PaymentProviderPaystack, "ref_123", amount, "NGN")

		assert.Equal(t, userID, deposit.UserID)
		assert.Equal(t, PaymentProviderPaystack, deposit.Provider)
		assert.Equal(t, "ref_123", deposit.ProviderReference)
		assert.Equal(t, PaymentTypeDeposit, deposit.PaymentType)
		assert.True(t, amount.Equal(deposit.Amount))
		assert.Equal(t, "NGN", deposit.CurrencyCode)
		assert.Equal(t, PaymentStatusPending, deposit.Status)
	})

	t.Run("CreateWithdrawalPayment", func(t *testing.T) {
		userID := uuid.New()
		amount := decimal.NewFromFloat(500)

		withdrawal := CreateWithdrawalPayment(userID, PaymentProviderFlutterwave, "ref_456", amount, "USD")

		assert.Equal(t, userID, withdrawal.UserID)
		assert.Equal(t, PaymentProviderFlutterwave, withdrawal.Provider)
		assert.Equal(t, "ref_456", withdrawal.ProviderReference)
		assert.Equal(t, PaymentTypeWithdrawal, withdrawal.PaymentType)
		assert.True(t, amount.Equal(withdrawal.Amount))
		assert.Equal(t, "USD", withdrawal.CurrencyCode)
		assert.Equal(t, PaymentStatusPending, withdrawal.Status)
	})
}
