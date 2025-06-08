package prediction

import (
	"time"

	"github.com/joefazee/neo/models"
	"github.com/shopspring/decimal"
)

// Config represents the configuration for the betting module
type Config struct {
	MaxBetAmount                    decimal.Decimal `env:"MAX_BET_AMOUNT"`
	MinBetAmount                    decimal.Decimal `env:"MIN_BET_AMOUNT"`
	MaxSlippagePercentage           decimal.Decimal `env:"MAX_SLIPPAGE_PERCENTAGE"`
	MaxPositionPerUser              decimal.Decimal `env:"MAX_POSITION_PER_USER"`
	MaxPositionPerMarket            decimal.Decimal `env:"MAX_POSITION_PER_MARKET"`
	BetTimeoutSeconds               int             `env:"BET_TIMEOUT_SECONDS"`
	EnableSlippageProtection        bool            `env:"ENABLE_SLIPPAGE_PROTECTION"`
	EnablePositionLimits            bool            `env:"ENABLE_POSITION_LIMITS"`
	EnableRealTimeUpdates           bool            `env:"ENABLE_REAL_TIME_UPDATES"`
	RequireKYCForBetting            bool            `env:"REQUIRE_KYC_FOR_BETTING"`
	MaxBetsPerMinute                int             `env:"MAX_BETS_PER_MINUTE"`
	MaxDailyBetAmount               decimal.Decimal `env:"MAX_DAILY_BET_AMOUNT"`
	CooldownPeriod                  time.Duration   `env:"COOLDOWN_PERIOD"`
	SignificantPriceImpactThreshold decimal.Decimal `env:"SIGNIFICANT_PRICE_IMPACT_THRESHOLD"`
	ModeratePriceImpactThreshold    decimal.Decimal `env:"MODERATE_PRICE_IMPACT_THRESHOLD"`
	HighPriceImpactThreshold        decimal.Decimal `env:"HIGH_PRICE_IMPACT_THRESHOLD"`
	MaxBetsForStatsCalculation      int             `env:"MAX_BETS_FOR_STATS_CALCULATION"`
	BetCancellationWindow           time.Duration   `env:"BET_CANCELLATION_WINDOW"`
}

func (c *Config) Validate() error {
	type validation struct {
		ok  bool
		err error
	}

	maxImpact := decimal.NewFromInt(100)

	checks := []validation{
		{c.MinBetAmount.GreaterThan(decimal.Zero), models.ErrInvalidBetAmountLimits},
		{c.MaxBetAmount.GreaterThan(c.MinBetAmount), models.ErrInvalidBetAmountLimits},

		{c.MaxSlippagePercentage.GreaterThanOrEqual(decimal.Zero) &&
			c.MaxSlippagePercentage.LessThanOrEqual(maxImpact),
			models.ErrInvalidSlippageLimit},

		{c.MaxPositionPerUser.GreaterThan(decimal.Zero), models.ErrInvalidPositionLimit},

		{c.BetTimeoutSeconds > 0 && c.BetTimeoutSeconds <= 300, models.ErrInvalidBetTimeout},
		{c.MaxBetsPerMinute > 0 && c.MaxBetsPerMinute <= 100, models.ErrInvalidRateLimit},

		{c.CooldownPeriod >= 0, models.ErrInvalidCooldownPeriod},
		{c.BetCancellationWindow >= 0, models.ErrInvalidBetCancellationWindow},

		{c.SignificantPriceImpactThreshold.GreaterThan(decimal.Zero) &&
			c.ModeratePriceImpactThreshold.GreaterThan(decimal.Zero) &&
			c.HighPriceImpactThreshold.GreaterThan(decimal.Zero),
			models.ErrInvalidPriceImpactThresholds},
		{c.SignificantPriceImpactThreshold.LessThanOrEqual(maxImpact) &&
			c.ModeratePriceImpactThreshold.LessThanOrEqual(maxImpact) &&
			c.HighPriceImpactThreshold.LessThanOrEqual(maxImpact),
			models.ErrInvalidPriceImpactThresholds},
	}

	for _, v := range checks {
		if !v.ok {
			return v.err
		}
	}
	return nil
}

// GetDefaultConfig returns the default betting configuration
func GetDefaultConfig() *Config {
	return &Config{
		MaxBetAmount:                    decimal.NewFromInt(50000),  // ₦50,000
		MinBetAmount:                    decimal.NewFromInt(100),    // ₦100
		MaxSlippagePercentage:           decimal.NewFromFloat(5.0),  // 5%
		MaxPositionPerUser:              decimal.NewFromInt(500000), // ₦500,000
		MaxPositionPerMarket:            decimal.NewFromInt(100000), // ₦100,000
		BetTimeoutSeconds:               30,                         // 30 seconds
		EnableSlippageProtection:        true,
		EnablePositionLimits:            true,
		EnableRealTimeUpdates:           true,
		RequireKYCForBetting:            true,
		MaxBetsPerMinute:                10,
		MaxDailyBetAmount:               decimal.NewFromInt(1000000), // ₦1,000,000
		CooldownPeriod:                  5 * time.Second,
		SignificantPriceImpactThreshold: decimal.NewFromFloat(5.0),  // 5% price impact
		ModeratePriceImpactThreshold:    decimal.NewFromFloat(2.0),  // 2% price impact
		HighPriceImpactThreshold:        decimal.NewFromFloat(10.0), // 10% price impact
		MaxBetsForStatsCalculation:      1000,
		BetCancellationWindow:           5 * time.Minute,
	}
}
