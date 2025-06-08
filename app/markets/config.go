package markets

import (
	"time"

	"github.com/joefazee/neo/models"
	"github.com/shopspring/decimal"
)

// Config represents the configuration for the markets module
type Config struct {
	DefaultRakePercentage      decimal.Decimal `env:"DEFAULT_RAKE_PERCENTAGE"`
	DefaultCreatorRevenueShare decimal.Decimal `env:"DEFAULT_CREATOR_REVENUE_SHARE"`
	MinQuorumAmount            decimal.Decimal `env:"MIN_QUORUM_AMOUNT"`
	HouseBotAmount             decimal.Decimal `env:"HOUSE_BOT_AMOUNT"`
	MaxBetAmount               decimal.Decimal `env:"MAX_BET_AMOUNT"`
	MinBetAmount               decimal.Decimal `env:"MIN_BET_AMOUNT"`
	MinMarketDuration          time.Duration   `env:"MIN_MARKET_DURATION"`
	MaxMarketDuration          time.Duration   `env:"MAX_MARKET_DURATION"`
	EnableSafeguards           bool            `env:"ENABLE_MARKET_SAFEGUARDS"`
	EnableHouseBot             bool            `env:"ENABLE_HOUSE_BOT"`
	EnableRealTimeUpdates      bool            `env:"ENABLE_REAL_TIME_UPDATES"`
	AutoResolveMarkets         bool            `env:"AUTO_RESOLVE_MARKETS"`
	MaxMarketsPerUser          int             `env:"MAX_MARKETS_PER_USER"`
	RequireModeration          bool            `env:"REQUIRE_MARKET_MODERATION"`
}

// Validate validates the market configuration
func (c *Config) Validate() error {
	if c.DefaultRakePercentage.LessThan(decimal.Zero) || c.DefaultRakePercentage.GreaterThan(decimal.NewFromFloat(0.2)) {
		return models.ErrInvalidMarketRake
	}

	if c.DefaultCreatorRevenueShare.LessThan(decimal.Zero) || c.DefaultCreatorRevenueShare.GreaterThan(decimal.NewFromInt(1)) {
		return models.ErrInvalidCreatorRevenueShare
	}

	if c.MinQuorumAmount.LessThanOrEqual(decimal.Zero) {
		return models.ErrInvalidMinQuorum
	}

	if c.HouseBotAmount.LessThanOrEqual(decimal.Zero) {
		return models.ErrInvalidHouseBotAmount
	}

	if c.MinBetAmount.LessThanOrEqual(decimal.Zero) || c.MaxBetAmount.LessThanOrEqual(c.MinBetAmount) {
		return models.ErrInvalidBetAmountLimits
	}

	if c.MinMarketDuration <= 0 || c.MaxMarketDuration <= c.MinMarketDuration {
		return models.ErrInvalidMarketDuration
	}

	if c.MaxMarketsPerUser <= 0 {
		return models.ErrInvalidMaxMarketsPerUser
	}

	return nil
}

// GetDefaultConfig returns the default configuration
func GetDefaultConfig() *Config {
	return &Config{
		DefaultRakePercentage:      decimal.NewFromFloat(0.05), // 5%
		DefaultCreatorRevenueShare: decimal.NewFromFloat(0.50), // 50% of rake
		MinQuorumAmount:            decimal.NewFromInt(5000),   // ₦5,000
		HouseBotAmount:             decimal.NewFromInt(10000),  // ₦10,000
		MaxBetAmount:               decimal.NewFromInt(50000),  // ₦50,000
		MinBetAmount:               decimal.NewFromInt(100),    // ₦100
		MinMarketDuration:          24 * time.Hour,             // 1 day
		MaxMarketDuration:          365 * 24 * time.Hour,       // 1 year
		EnableSafeguards:           true,
		EnableHouseBot:             true,
		EnableRealTimeUpdates:      true,
		AutoResolveMarkets:         false,
		MaxMarketsPerUser:          10,
		RequireModeration:          true,
	}
}
