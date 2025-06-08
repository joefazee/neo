package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// PaymentProvider represents supported payment providers
type PaymentProvider string

const (
	PaymentProviderPaystack    PaymentProvider = "paystack"
	PaymentProviderFlutterwave PaymentProvider = "flutterwave"
	PaymentProviderMonnify     PaymentProvider = "monnify"
)

// PaymentType represents the type of payment
type PaymentType string

const (
	PaymentTypeDeposit    PaymentType = "deposit"
	PaymentTypeWithdrawal PaymentType = "withdrawal"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusSuccess    PaymentStatus = "success"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusCancelled  PaymentStatus = "canceled"
)

// ProviderResponse represents the payment provider's response data
type ProviderResponse struct {
	TransactionID string                 `json:"transaction_id,omitempty"`
	Reference     string                 `json:"reference,omitempty"`
	Status        string                 `json:"status,omitempty"`
	Message       string                 `json:"message,omitempty"`
	Gateway       string                 `json:"gateway,omitempty"`
	Channel       string                 `json:"channel,omitempty"`
	Currency      string                 `json:"currency,omitempty"`
	Amount        decimal.Decimal        `json:"amount,omitempty"`
	Fees          decimal.Decimal        `json:"fees,omitempty"`
	Authorization map[string]interface{} `json:"authorization,omitempty"`
	Customer      map[string]interface{} `json:"customer,omitempty"`
	PaymentDate   *time.Time             `json:"payment_date,omitempty"`
	RawResponse   map[string]interface{} `json:"raw_response,omitempty"`
}

// Value implements driver.Valuer interface
func (pr *ProviderResponse) Value() (driver.Value, error) {
	return json.Marshal(pr)
}

// Scan implements sql.Scanner interface
func (pr *ProviderResponse) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, pr)
	case string:
		return json.Unmarshal([]byte(v), pr)
	}
	return nil
}

// PaymentTransaction represents a payment transaction with external providers
type PaymentTransaction struct {
	ID                uuid.UUID        `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID            uuid.UUID        `gorm:"type:uuid;not null;index:idx_payment_transactions_user" json:"user_id"`
	TransactionID     *uuid.UUID       `gorm:"type:uuid" json:"transaction_id"`
	Provider          PaymentProvider  `gorm:"type:varchar(20);not null;index:idx_payment_transactions_provider_ref" json:"provider"`
	ProviderReference string           `gorm:"type:varchar(100);not null;index:idx_payment_transactions_provider_ref" json:"provider_reference"`
	PaymentType       PaymentType      `gorm:"type:varchar(20);not null" json:"payment_type"`
	Amount            decimal.Decimal  `gorm:"type:decimal(20,2);not null" json:"amount"`
	CurrencyCode      string           `gorm:"type:varchar(3);not null" json:"currency_code"`
	Status            PaymentStatus    `gorm:"type:varchar(20);default:'pending'" json:"status"`
	ProviderResponse  ProviderResponse `gorm:"type:jsonb;default:'{}'" json:"provider_response"`
	WebhookVerified   bool             `gorm:"default:false" json:"webhook_verified"`
	CreatedAt         time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time        `gorm:"autoUpdateTime" json:"updated_at"`

	// Associations
	User        *User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Transaction *Transaction `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
}

// TableName specifies the table name for PaymentTransaction model
func (*PaymentTransaction) TableName() string {
	return "payment_transactions"
}

// BeforeCreate sets up the model before creation
func (pt *PaymentTransaction) BeforeCreate(_ *gorm.DB) error {
	if pt.ID == uuid.Nil {
		pt.ID = uuid.New()
	}
	return nil
}

// IsDeposit checks if this is a deposit transaction
func (pt *PaymentTransaction) IsDeposit() bool {
	return pt.PaymentType == PaymentTypeDeposit
}

// IsWithdrawal checks if this is a withdrawal transaction
func (pt *PaymentTransaction) IsWithdrawal() bool {
	return pt.PaymentType == PaymentTypeWithdrawal
}

// IsPending checks if the payment is still pending
func (pt *PaymentTransaction) IsPending() bool {
	return pt.Status == PaymentStatusPending
}

// IsProcessing checks if the payment is being processed
func (pt *PaymentTransaction) IsProcessing() bool {
	return pt.Status == PaymentStatusProcessing
}

// IsSuccessful checks if the payment was successful
func (pt *PaymentTransaction) IsSuccessful() bool {
	return pt.Status == PaymentStatusSuccess
}

// IsFailed checks if the payment failed
func (pt *PaymentTransaction) IsFailed() bool {
	return pt.Status == PaymentStatusFailed
}

// IsCancelled checks if the payment was canceled
func (pt *PaymentTransaction) IsCancelled() bool {
	return pt.Status == PaymentStatusCancelled
}

// IsCompleted checks if the payment is in a final state
func (pt *PaymentTransaction) IsCompleted() bool {
	return pt.IsSuccessful() || pt.IsFailed() || pt.IsCancelled()
}

// UpdateStatus updates the payment status and provider response
func (pt *PaymentTransaction) UpdateStatus(status PaymentStatus, response *ProviderResponse) {
	pt.Status = status
	pt.ProviderResponse = *response
}

// MarkWebhookVerified marks the webhook as verified
func (pt *PaymentTransaction) MarkWebhookVerified() {
	pt.WebhookVerified = true
}

// GetProviderFees returns the fees charged by the payment provider
func (pt *PaymentTransaction) GetProviderFees() decimal.Decimal {
	return pt.ProviderResponse.Fees
}

// GetNetAmount returns the net amount after provider fees
func (pt *PaymentTransaction) GetNetAmount() decimal.Decimal {
	return pt.Amount.Sub(pt.GetProviderFees())
}

// Validate performs validation on the payment transaction model
func (pt *PaymentTransaction) Validate() error {
	if pt.UserID == uuid.Nil {
		return ErrInvalidUserID
	}
	if pt.Provider == "" {
		return ErrInvalidPaymentProvider
	}
	if pt.ProviderReference == "" {
		return ErrInvalidProviderReference
	}
	if pt.Amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidTransactionAmount
	}
	if len(pt.CurrencyCode) != 3 {
		return ErrInvalidCurrencyCode
	}
	return nil
}

// CreateDepositPayment creates a new deposit payment transaction
func CreateDepositPayment(userID uuid.UUID,
	provider PaymentProvider,
	reference string,
	amount decimal.Decimal,
	currency string) *PaymentTransaction {
	return &PaymentTransaction{
		UserID:            userID,
		Provider:          provider,
		ProviderReference: reference,
		PaymentType:       PaymentTypeDeposit,
		Amount:            amount,
		CurrencyCode:      currency,
		Status:            PaymentStatusPending,
	}
}

// CreateWithdrawalPayment creates a new withdrawal payment transaction
func CreateWithdrawalPayment(userID uuid.UUID,
	provider PaymentProvider,
	reference string,
	amount decimal.Decimal,
	currency string) *PaymentTransaction {
	return &PaymentTransaction{
		UserID:            userID,
		Provider:          provider,
		ProviderReference: reference,
		PaymentType:       PaymentTypeWithdrawal,
		Amount:            amount,
		CurrencyCode:      currency,
		Status:            PaymentStatusPending,
	}
}
