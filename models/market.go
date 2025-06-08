package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// MarketType represents the type of market
type MarketType string

const (
	MarketTypeBinary       MarketType = "binary"
	MarketTypeMultiOutcome MarketType = "multi_outcome"
)

// MarketStatus represents the current status of a market
type MarketStatus string

const (
	MarketStatusDraft    MarketStatus = "draft"
	MarketStatusOpen     MarketStatus = "open"
	MarketStatusClosed   MarketStatus = "closed"
	MarketStatusResolved MarketStatus = "resolved"
	MarketStatusVoided   MarketStatus = "voided"
)

// SafeguardConfig represents market safeguard configuration
type SafeguardConfig struct {
	MinQuorumAmount    decimal.Decimal `json:"min_quorum_amount"`
	MinOutcomes        int             `json:"min_outcomes"`
	HouseBotEnabled    bool            `json:"house_bot_enabled"`
	HouseBotAmount     decimal.Decimal `json:"house_bot_amount"`
	ImbalanceThreshold decimal.Decimal `json:"imbalance_threshold"`
	VoidOnQuorumFail   bool            `json:"void_on_quorum_fail"`
}

// OracleConfig represents oracle configuration for market resolution
type OracleConfig struct {
	Provider       string            `json:"provider"`
	DataSource     string            `json:"data_source"`
	ResolutionURL  string            `json:"resolution_url"`
	Criteria       map[string]string `json:"criteria"`
	AutoResolve    bool              `json:"auto_resolve"`
	BackupProvider string            `json:"backup_provider,omitempty"`
}

// MarketMetadata represents additional market metadata
type MarketMetadata struct {
	Tags          []string   `json:"tags,omitempty"`
	ImageURL      string     `json:"image_url,omitempty"`
	SourceURL     string     `json:"source_url,omitempty"`
	FeaturedUntil *time.Time `json:"featured_until,omitempty"`
	ViewCount     int64      `json:"view_count,omitempty"`
}

// Value implementations for JSONB fields
func (s *SafeguardConfig) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *SafeguardConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	}
	return nil
}

func (o *OracleConfig) Value() (driver.Value, error) {
	return json.Marshal(o)
}

func (o *OracleConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, o)
	case string:
		return json.Unmarshal([]byte(v), o)
	}
	return nil
}

func (m *MarketMetadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *MarketMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, m)
	case string:
		return json.Unmarshal([]byte(v), m)
	}
	return nil
}

// Market represents a prediction market
type Market struct {
	ID                  uuid.UUID        `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	CountryID           uuid.UUID        `gorm:"type:uuid;not null;index:idx_markets_country_category" json:"country_id"`
	CategoryID          uuid.UUID        `gorm:"type:uuid;not null;index:idx_markets_country_category" json:"category_id"`
	CreatorID           *uuid.UUID       `gorm:"type:uuid;index" json:"creator_id"`
	Title               string           `gorm:"type:varchar(255);not null" json:"title"`
	Description         string           `gorm:"type:text;not null" json:"description"`
	MarketType          MarketType       `gorm:"type:varchar(20);default:'binary'" json:"market_type"`
	Status              MarketStatus     `gorm:"type:varchar(20);default:'draft';index" json:"status"`
	CloseTime           time.Time        `gorm:"type:timestamptz;not null;index" json:"close_time"`
	ResolutionDeadline  time.Time        `gorm:"type:timestamptz;not null" json:"resolution_deadline"`
	ResolvedAt          *time.Time       `gorm:"type:timestamptz" json:"resolved_at"`
	ResolvedOutcome     string           `gorm:"type:varchar(100)" json:"resolved_outcome"`
	ResolutionSource    string           `gorm:"type:text" json:"resolution_source"`
	MinBetAmount        decimal.Decimal  `gorm:"type:decimal(20,2);not null;default:100.00" json:"min_bet_amount"`
	MaxBetAmount        *decimal.Decimal `gorm:"type:decimal(20,2)" json:"max_bet_amount"`
	TotalPoolAmount     decimal.Decimal  `gorm:"type:decimal(20,2);default:0.00" json:"total_pool_amount"`
	RakePercentage      decimal.Decimal  `gorm:"type:decimal(5,4);default:0.0500" json:"rake_percentage"`
	CreatorRevenueShare decimal.Decimal  `gorm:"type:decimal(5,4);default:0.5000" json:"creator_revenue_share"`
	SafeguardConfig     SafeguardConfig  `gorm:"type:jsonb;default:'{}'" json:"safeguard_config"`
	OracleConfig        OracleConfig     `gorm:"type:jsonb;default:'{}'" json:"oracle_config"`
	Metadata            MarketMetadata   `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt           time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time        `gorm:"autoUpdateTime" json:"updated_at"`

	// Associations
	Country     *Country        `gorm:"foreignKey:CountryID" json:"country,omitempty"`
	Category    *Category       `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Creator     *User           `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
	Outcomes    []MarketOutcome `gorm:"foreignKey:MarketID;constraint:OnDelete:CASCADE" json:"outcomes,omitempty"`
	Bets        []Bet           `gorm:"foreignKey:MarketID" json:"-"`
	Settlements []Settlement    `gorm:"foreignKey:MarketID" json:"-"`
}

// TableName specifies the table name for Market model
func (*Market) TableName() string {
	return "markets"
}

// BeforeCreate sets up the model before creation
func (m *Market) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// IsOpen checks if the market is open for betting
func (m *Market) IsOpen() bool {
	return m.Status == MarketStatusOpen && time.Now().Before(m.CloseTime)
}

// IsClosed checks if the market is closed
func (m *Market) IsClosed() bool {
	return m.Status == MarketStatusClosed || time.Now().After(m.CloseTime)
}

// IsResolved checks if the market has been resolved
func (m *Market) IsResolved() bool {
	return m.Status == MarketStatusResolved && m.ResolvedAt != nil
}

// IsVoided checks if the market has been voided
func (m *Market) IsVoided() bool {
	return m.Status == MarketStatusVoided
}

// CanBet checks if betting is allowed on this market
func (m *Market) CanBet() bool {
	return m.IsOpen() && !m.IsVoided()
}

// CanResolve checks if the market can be resolved
func (m *Market) CanResolve() bool {
	return m.IsClosed() && !m.IsResolved() && !m.IsVoided()
}

// GetRakeAmount calculates the rake amount for a given bet amount
func (m *Market) GetRakeAmount(betAmount decimal.Decimal) decimal.Decimal {
	return betAmount.Mul(m.RakePercentage)
}

// GetCreatorFee calculates the creator fee for a given rake amount
func (m *Market) GetCreatorFee(rakeAmount decimal.Decimal) decimal.Decimal {
	return rakeAmount.Mul(m.CreatorRevenueShare)
}

// HasMinQuorum checks if the market meets minimum quorum requirements
func (m *Market) HasMinQuorum() bool {
	if m.SafeguardConfig.MinQuorumAmount.IsZero() {
		return true // No quorum requirement
	}
	return m.TotalPoolAmount.GreaterThanOrEqual(m.SafeguardConfig.MinQuorumAmount)
}

// ValidateBetAmount checks if a bet amount is within allowed limits
func (m *Market) ValidateBetAmount(amount decimal.Decimal) error {
	if amount.LessThan(m.MinBetAmount) {
		return ErrBetTooSmall
	}
	if m.MaxBetAmount != nil && amount.GreaterThan(*m.MaxBetAmount) {
		return ErrBetTooLarge
	}
	return nil
}

// Resolve resolves the market with the given outcome
func (m *Market) Resolve(outcome, source string) error {
	if !m.CanResolve() {
		return ErrMarketNotOpen
	}

	now := time.Now()
	m.Status = MarketStatusResolved
	m.ResolvedOutcome = outcome
	m.ResolutionSource = source
	m.ResolvedAt = &now

	return nil
}

// Void voids the market
func (m *Market) Void() error {
	if m.IsResolved() {
		return ErrMarketAlreadyClosed
	}

	m.Status = MarketStatusVoided
	return nil
}

// Validate performs validation on the market model
func (m *Market) Validate() error {
	if m.CountryID == uuid.Nil {
		return ErrInvalidCountryID
	}
	if m.CategoryID == uuid.Nil {
		return ErrInvalidCategoryName
	}
	if m.Title == "" {
		return ErrInvalidMarketTitle
	}
	if m.CloseTime.Before(time.Now()) {
		return ErrInvalidCloseTime
	}
	if m.ResolutionDeadline.Before(m.CloseTime) {
		return ErrInvalidResolutionTime
	}
	if m.MinBetAmount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidBetAmount
	}
	return nil
}
