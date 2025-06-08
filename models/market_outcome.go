package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// MarketOutcome represents a possible outcome for a market
type MarketOutcome struct {
	ID               uuid.UUID       `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	MarketID         uuid.UUID       `gorm:"type:uuid;not null;index:idx_market_outcomes_market" json:"market_id"`
	OutcomeKey       string          `gorm:"type:varchar(50);not null" json:"outcome_key"` // 'yes', 'no', 'openai', etc.
	OutcomeLabel     string          `gorm:"type:varchar(100);not null" json:"outcome_label"`
	SortOrder        int             `gorm:"default:0" json:"sort_order"`
	PoolAmount       decimal.Decimal `gorm:"type:decimal(20,2);default:0.00" json:"pool_amount"`
	IsWinningOutcome *bool           `gorm:"type:boolean" json:"is_winning_outcome"`
	CreatedAt        time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time       `gorm:"autoUpdateTime" json:"updated_at"`

	// Associations
	Market *Market `gorm:"foreignKey:MarketID;constraint:OnDelete:CASCADE" json:"market,omitempty"`
	Bets   []Bet   `gorm:"foreignKey:MarketOutcomeID" json:"-"`
}

// TableName specifies the table name for MarketOutcome model
func (*MarketOutcome) TableName() string {
	return "market_outcomes"
}

// BeforeCreate sets up the model before creation
func (mo *MarketOutcome) BeforeCreate(_ *gorm.DB) error {
	if mo.ID == uuid.Nil {
		mo.ID = uuid.New()
	}
	return nil
}

// GetCurrentPrice calculates the current price based on pool distribution
func (mo *MarketOutcome) GetCurrentPrice(totalMarketPool decimal.Decimal) decimal.Decimal {
	if totalMarketPool.IsZero() {
		return decimal.NewFromInt(50) // Default 50% if no bets
	}

	percentage := mo.PoolAmount.Div(totalMarketPool).Mul(decimal.NewFromInt(100))

	// Ensure price is between 1 and 99
	if percentage.LessThan(decimal.NewFromInt(1)) {
		return decimal.NewFromInt(1)
	}
	if percentage.GreaterThan(decimal.NewFromInt(99)) {
		return decimal.NewFromInt(99)
	}

	return percentage
}

// AddToPool adds the specified amount to this outcome's pool
func (mo *MarketOutcome) AddToPool(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidBetAmount
	}
	mo.PoolAmount = mo.PoolAmount.Add(amount)
	return nil
}

// SetAsWinner marks this outcome as the winning outcome
func (mo *MarketOutcome) SetAsWinner() {
	winner := true
	mo.IsWinningOutcome = &winner
}

// SetAsLoser marks this outcome as a losing outcome
func (mo *MarketOutcome) SetAsLoser() {
	loser := false
	mo.IsWinningOutcome = &loser
}

// IsWinner checks if this outcome is the winning outcome
func (mo *MarketOutcome) IsWinner() bool {
	return mo.IsWinningOutcome != nil && *mo.IsWinningOutcome
}

// IsLoser checks if this outcome is a losing outcome
func (mo *MarketOutcome) IsLoser() bool {
	return mo.IsWinningOutcome != nil && !*mo.IsWinningOutcome
}

// IsUnresolved checks if this outcome's result is still unknown
func (mo *MarketOutcome) IsUnresolved() bool {
	return mo.IsWinningOutcome == nil
}

// Validate performs validation on the market outcome model
func (mo *MarketOutcome) Validate() error {
	if mo.MarketID == uuid.Nil {
		return ErrInvalidMarketID
	}
	if mo.OutcomeKey == "" {
		return ErrInvalidOutcomeKey
	}
	if mo.OutcomeLabel == "" {
		return ErrInvalidOutcomeLabel
	}
	if mo.PoolAmount.LessThan(decimal.Zero) {
		return ErrInvalidBetAmount
	}
	return nil
}

// GetBetCount returns the number of bets placed on this outcome
func (mo *MarketOutcome) GetBetCount(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Model(&Bet{}).Where("market_outcome_id = ? AND status = ?", mo.ID, "active").Count(&count).Error
	return count, err
}

// GetUniqueBettors returns the number of unique bettors on this outcome
func (mo *MarketOutcome) GetUniqueBettors(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Model(&Bet{}).Where("market_outcome_id = ? AND status = ?", mo.ID, "active").
		Distinct("user_id").Count(&count).Error
	return count, err
}
