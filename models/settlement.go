package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// SettlementType represents the type of settlement
type SettlementType string

const (
	SettlementTypeWin    SettlementType = "win"
	SettlementTypeLoss   SettlementType = "loss"
	SettlementTypeRefund SettlementType = "refund"
)

// Settlement represents the settlement of a bet (immutable record)
type Settlement struct {
	ID             uuid.UUID       `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	MarketID       uuid.UUID       `gorm:"type:uuid;not null;index:idx_settlements_market" json:"market_id"`
	UserID         uuid.UUID       `gorm:"type:uuid;not null;index:idx_settlements_user" json:"user_id"`
	BetID          uuid.UUID       `gorm:"type:uuid;not null" json:"bet_id"`
	SettlementType SettlementType  `gorm:"type:varchar(20);not null" json:"settlement_type"`
	OriginalAmount decimal.Decimal `gorm:"type:decimal(20,2);not null" json:"original_amount"`
	PayoutAmount   decimal.Decimal `gorm:"type:decimal(20,2);not null;default:0.00" json:"payout_amount"`
	RakeAmount     decimal.Decimal `gorm:"type:decimal(20,2);not null;default:0.00" json:"rake_amount"`
	TransactionID  *uuid.UUID      `gorm:"type:uuid" json:"transaction_id"`
	CreatedAt      time.Time       `gorm:"autoCreateTime" json:"created_at"`

	// Associations (Note: Settlements are immutable, no updates)
	Market      *Market      `gorm:"foreignKey:MarketID" json:"market,omitempty"`
	User        *User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Bet         *Bet         `gorm:"foreignKey:BetID" json:"bet,omitempty"`
	Transaction *Transaction `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
}

// TableName specifies the table name for Settlement model
func (*Settlement) TableName() string {
	return "settlements"
}

// BeforeCreate sets up the model before creation
func (s *Settlement) BeforeCreate(_ *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// IsWin checks if this is a winning settlement
func (s *Settlement) IsWin() bool {
	return s.SettlementType == SettlementTypeWin
}

// IsLoss checks if this is a losing settlement
func (s *Settlement) IsLoss() bool {
	return s.SettlementType == SettlementTypeLoss
}

// IsRefund checks if this is a refund settlement
func (s *Settlement) IsRefund() bool {
	return s.SettlementType == SettlementTypeRefund
}

// GetNetAmount returns the net amount (payout - original bet)
func (s *Settlement) GetNetAmount() decimal.Decimal {
	return s.PayoutAmount.Sub(s.OriginalAmount)
}

// GetReturnMultiple calculates the return multiple
func (s *Settlement) GetReturnMultiple() decimal.Decimal {
	if s.OriginalAmount.IsZero() {
		return decimal.Zero
	}
	return s.PayoutAmount.Div(s.OriginalAmount)
}

// Validate performs validation on the settlement model
func (s *Settlement) Validate() error {
	if s.MarketID == uuid.Nil {
		return ErrInvalidMarketID
	}
	if s.UserID == uuid.Nil {
		return ErrInvalidUserID
	}
	if s.BetID == uuid.Nil {
		return ErrInvalidBetAmount
	}
	if s.OriginalAmount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidBetAmount
	}
	if s.PayoutAmount.LessThan(decimal.Zero) {
		return ErrInvalidTransactionAmount
	}
	if s.RakeAmount.LessThan(decimal.Zero) {
		return ErrInvalidTransactionAmount
	}
	return nil
}

// CreateWinSettlement creates a winning settlement
func CreateWinSettlement(marketID,
	userID, betID uuid.UUID,
	originalAmount, payoutAmount,
	rakeAmount decimal.Decimal) *Settlement {
	return &Settlement{
		MarketID:       marketID,
		UserID:         userID,
		BetID:          betID,
		SettlementType: SettlementTypeWin,
		OriginalAmount: originalAmount,
		PayoutAmount:   payoutAmount,
		RakeAmount:     rakeAmount,
	}
}

// CreateLossSettlement creates a losing settlement
func CreateLossSettlement(marketID, userID, betID uuid.UUID, originalAmount decimal.Decimal) *Settlement {
	return &Settlement{
		MarketID:       marketID,
		UserID:         userID,
		BetID:          betID,
		SettlementType: SettlementTypeLoss,
		OriginalAmount: originalAmount,
		PayoutAmount:   decimal.Zero,
		RakeAmount:     decimal.Zero,
	}
}

// CreateRefundSettlement creates a refund settlement
func CreateRefundSettlement(marketID, userID, betID uuid.UUID, originalAmount decimal.Decimal) *Settlement {
	return &Settlement{
		MarketID:       marketID,
		UserID:         userID,
		BetID:          betID,
		SettlementType: SettlementTypeRefund,
		OriginalAmount: originalAmount,
		PayoutAmount:   originalAmount, // Full refund
		RakeAmount:     decimal.Zero,
	}
}
