package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// CountryConfig represents country-specific configuration
type CountryConfig struct {
	ContractUnit decimal.Decimal `json:"contract_unit"`
	MinBet       decimal.Decimal `json:"min_bet"`
	MaxBet       decimal.Decimal `json:"max_bet"`
	KYCRequired  bool            `json:"kyc_required"`
	TaxRate      decimal.Decimal `json:"tax_rate,omitempty"`
	Timezone     string          `json:"timezone,omitempty"`
}

// Value implements driver.Valuer interface for database storage
func (c *CountryConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements sql.Scanner interface for database retrieval
func (c *CountryConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, c)
	case string:
		return json.Unmarshal([]byte(v), c)
	}
	return nil
}

// Country represents a country/region in the system
type Country struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name           string         `gorm:"type:varchar(100);not null;unique" json:"name"`
	Code           string         `gorm:"type:varchar(3);not null;unique" json:"code"`   // ISO 3166-1 alpha-3
	CurrencyCode   string         `gorm:"type:varchar(3);not null" json:"currency_code"` // ISO 4217
	CurrencySymbol string         `gorm:"type:varchar(10);not null" json:"currency_symbol"`
	IsActive       *bool          `gorm:"default:true" json:"is_active"`
	Config         *CountryConfig `gorm:"type:jsonb;default:'{}'" json:"config"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	// Associations
	Categories []Category `gorm:"foreignKey:CountryID;constraint:OnDelete:CASCADE" json:"categories,omitempty"`
	Users      []User     `gorm:"foreignKey:CountryID" json:"-"`
	Markets    []Market   `gorm:"foreignKey:CountryID" json:"-"`
}

// TableName specifies the table name for Country model
func (*Country) TableName() string {
	return "countries"
}

// BeforeCreate sets up the model before creation
func (c *Country) BeforeCreate(_ *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// IsValidCurrency checks if the currency code is valid
func (c *Country) IsValidCurrency() bool {
	return len(c.CurrencyCode) == 3 && c.CurrencyCode != ""
}

// GetMinBetAmount returns the minimum bet amount for this country
func (c *Country) GetMinBetAmount() decimal.Decimal {
	if !c.Config.MinBet.IsZero() {
		return c.Config.MinBet
	}
	return decimal.NewFromInt(100)
}

// GetMaxBetAmount returns the maximum bet amount for this country
func (c *Country) GetMaxBetAmount() decimal.Decimal {
	if !c.Config.MaxBet.IsZero() {
		return c.Config.MaxBet
	}
	return decimal.NewFromInt(50000)
}

// GetContractUnit returns the contract unit value for this country
func (c *Country) GetContractUnit() decimal.Decimal {
	if !c.Config.ContractUnit.IsZero() {
		return c.Config.ContractUnit
	}
	return decimal.NewFromInt(100)
}

// RequiresKYC returns whether KYC is required for this country
func (c *Country) RequiresKYC() bool {
	return c.Config.KYCRequired
}

func (c *Country) IsActiveValue() bool {
	return c.IsActive != nil && *c.IsActive
}

// Validate performs validation on the country model
func (c *Country) Validate() error {
	if c.Name == "" {
		return ErrInvalidCountryName
	}
	if len(c.Code) != 3 {
		return ErrInvalidCountryCode
	}
	if !c.IsValidCurrency() {
		return ErrInvalidCurrencyCode
	}
	if c.CurrencySymbol == "" {
		return ErrInvalidCurrencySymbol
	}
	return nil
}
