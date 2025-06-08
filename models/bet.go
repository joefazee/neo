package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// BetStatus represents the status of a bet
type BetStatus string

const (
	BetStatusActive   BetStatus = "active"
	BetStatusSettled  BetStatus = "settled"
	BetStatusRefunded BetStatus = "refunded"
)

// BetMetadata represents additional bet metadata
type BetMetadata struct {
	IPAddress       string          `json:"ip_address,omitempty"`
	UserAgent       string          `json:"user_agent,omitempty"`
	SlippageWarning bool            `json:"slippage_warning,omitempty"`
	PriceAtTime     decimal.Decimal `json:"price_at_time,omitempty"`
}

// Value implements driver.Valuer interface
func (bm *BetMetadata) Value() (driver.Value, error) {
	return json.Marshal(bm)
}

// Scan implements sql.Scanner interface
func (bm *BetMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, bm)
	case string:
		return json.Unmarshal([]byte(v), bm)
	}
	return nil
}

// Bet represents a user's bet on a market outcome
type Bet struct {
	ID               uuid.UUID        `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID           uuid.UUID        `gorm:"type:uuid;not null;index:idx_bets_user" json:"user_id"`
	MarketID         uuid.UUID        `gorm:"type:uuid;not null;index:idx_bets_market" json:"market_id"`
	MarketOutcomeID  uuid.UUID        `gorm:"type:uuid;not null" json:"market_outcome_id"`
	Amount           decimal.Decimal  `gorm:"type:decimal(20,2);not null;check:amount > 0" json:"amount"`
	ContractsBought  decimal.Decimal  `gorm:"type:decimal(20,8);not null;check:contracts_bought > 0" json:"contracts_bought"`
	PricePerContract decimal.Decimal  `gorm:"type:decimal(20,2);not null;check:price_per_contract > 0" json:"price_per_contract"`
	TotalCost        decimal.Decimal  `gorm:"type:decimal(20,2);not null;check:total_cost > 0" json:"total_cost"`
	TransactionID    uuid.UUID        `gorm:"type:uuid;not null" json:"transaction_id"`
	Status           BetStatus        `gorm:"type:varchar(20);default:'active';index" json:"status"`
	SettledAt        *time.Time       `gorm:"type:timestamptz" json:"settled_at"`
	SettlementAmount *decimal.Decimal `gorm:"type:decimal(20,2)" json:"settlement_amount"`
	Metadata         *BetMetadata     `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt        time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time        `gorm:"autoUpdateTime" json:"updated_at"`

	// Associations
	User          *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Market        *Market        `gorm:"foreignKey:MarketID" json:"market,omitempty"`
	MarketOutcome *MarketOutcome `gorm:"foreignKey:MarketOutcomeID" json:"market_outcome,omitempty"`
	Transaction   *Transaction   `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
	Settlements   []Settlement   `gorm:"foreignKey:BetID" json:"-"`
}

// TableName specifies the table name for Bet model
func (*Bet) TableName() string {
	return "bets"
}

// BeforeCreate sets up the model before creation
func (b *Bet) BeforeCreate(_ *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// IsActive checks if the bet is still active
func (b *Bet) IsActive() bool {
	return b.Status == BetStatusActive
}

// IsSettled checks if the bet has been settled
func (b *Bet) IsSettled() bool {
	return b.Status == BetStatusSettled && b.SettledAt != nil
}

// IsRefunded checks if the bet has been refunded
func (b *Bet) IsRefunded() bool {
	return b.Status == BetStatusRefunded
}

// CalculatePotentialPayout calculates potential payout if this outcome wins
func (b *Bet) CalculatePotentialPayout(totalWinningContracts, prizePool decimal.Decimal) decimal.Decimal {
	if totalWinningContracts.IsZero() {
		return decimal.Zero
	}

	// Payout = (user's contracts / total winning contracts) * prize pool
	return b.ContractsBought.Div(totalWinningContracts).Mul(prizePool)
}

// Settle settles the bet with the given settlement amount
func (b *Bet) Settle(settlementAmount decimal.Decimal) error {
	if !b.IsActive() {
		return ErrBetAlreadySettled
	}

	now := time.Now()
	b.Status = BetStatusSettled
	b.SettledAt = &now
	b.SettlementAmount = &settlementAmount

	return nil
}

// Refund refunds the bet
func (b *Bet) Refund() error {
	if !b.IsActive() {
		return ErrBetAlreadySettled
	}

	now := time.Now()
	b.Status = BetStatusRefunded
	b.SettledAt = &now
	refundAmount := b.Amount // Full refund of original amount
	b.SettlementAmount = &refundAmount

	return nil
}

// GetProfitLoss calculates the profit or loss for this bet
func (b *Bet) GetProfitLoss() decimal.Decimal {
	if !b.IsSettled() {
		return decimal.Zero
	}

	if b.SettlementAmount == nil {
		return b.Amount.Neg() // Total loss
	}

	return b.SettlementAmount.Sub(b.Amount)
}

// GetReturn calculates the return multiple for this bet
func (b *Bet) GetReturn() decimal.Decimal {
	if !b.IsSettled() || b.SettlementAmount == nil {
		return decimal.Zero
	}

	if b.Amount.IsZero() {
		return decimal.Zero
	}

	return b.SettlementAmount.Div(b.Amount)
}

// Validate performs validation on the bet model
func (b *Bet) Validate() error {
	if b.UserID == uuid.Nil {
		return ErrInvalidUserID
	}
	if b.MarketID == uuid.Nil {
		return ErrInvalidMarketID
	}
	if b.MarketOutcomeID == uuid.Nil {
		return ErrInvalidOutcomeKey
	}
	if b.Amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidBetAmount
	}
	if b.ContractsBought.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidBetAmount
	}
	if b.PricePerContract.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidBetAmount
	}
	if b.TotalCost.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidBetAmount
	}
	if b.TransactionID == uuid.Nil {
		return ErrInvalidTransactionType
	}
	return nil
}
