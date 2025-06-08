package prediction

import (
	"testing"
	"time"

	"github.com/joefazee/neo/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	assert.NotNil(t, config, "Default config should not be nil")

	assert.True(t, config.MaxBetAmount.Equal(decimal.NewFromInt(50000)), "Default MaxBetAmount mismatch")
	assert.True(t, config.MinBetAmount.Equal(decimal.NewFromInt(100)), "Default MinBetAmount mismatch")
	assert.True(t, config.MaxSlippagePercentage.Equal(decimal.NewFromFloat(5.0)), "Default MaxSlippagePercentage mismatch")
	assert.Equal(t, 30, config.BetTimeoutSeconds, "Default BetTimeoutSeconds mismatch")
	assert.True(t, config.EnableSlippageProtection, "Default EnableSlippageProtection mismatch")
	assert.True(t, config.EnablePositionLimits, "Default EnablePositionLimits mismatch")
	assert.True(t, config.RequireKYCForBetting, "Default RequireKYCForBetting mismatch")
	assert.Equal(t, 10, config.MaxBetsPerMinute, "Default MaxBetsPerMinute mismatch")
	assert.True(t, config.MaxDailyBetAmount.Equal(decimal.NewFromInt(1000000)), "Default MaxDailyBetAmount mismatch")
	assert.Equal(t, 5*time.Second, config.CooldownPeriod, "Default CooldownPeriod mismatch")

	err := config.Validate()
	assert.NoError(t, err, "Default configuration should be valid")
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		modifier    func(c *Config)
		expectedErr error
	}{
		{
			name:        "Valid default configuration",
			modifier:    func(_ *Config) {},
			expectedErr: nil,
		},

		{
			name: "Invalid MinBetAmount (zero)",
			modifier: func(c *Config) {
				c.MinBetAmount = decimal.Zero
			},
			expectedErr: models.ErrInvalidBetAmountLimits,
		},
		{
			name: "Invalid MinBetAmount (negative)",
			modifier: func(c *Config) {
				c.MinBetAmount = decimal.NewFromFloat(-10)
			},
			expectedErr: models.ErrInvalidBetAmountLimits,
		},

		{
			name: "Invalid MaxBetAmount (less than MinBetAmount)",
			modifier: func(c *Config) {
				c.MinBetAmount = decimal.NewFromInt(100)
				c.MaxBetAmount = decimal.NewFromInt(50)
			},
			expectedErr: models.ErrInvalidBetAmountLimits,
		},
		{
			name: "Invalid MaxBetAmount (equal to MinBetAmount)",
			modifier: func(c *Config) {
				c.MinBetAmount = decimal.NewFromInt(100)
				c.MaxBetAmount = decimal.NewFromInt(100)
			},
			expectedErr: models.ErrInvalidBetAmountLimits,
		},
		{
			name: "Invalid MaxSlippagePercentage (negative)",
			modifier: func(c *Config) {
				c.MaxSlippagePercentage = decimal.NewFromFloat(-1.0)
			},
			expectedErr: models.ErrInvalidSlippageLimit,
		},
		{
			name: "Invalid MaxSlippagePercentage (greater than 100)",
			modifier: func(c *Config) {
				c.MaxSlippagePercentage = decimal.NewFromFloat(101.0)
			},
			expectedErr: models.ErrInvalidSlippageLimit,
		},
		{
			name: "Valid MaxSlippagePercentage (zero)",
			modifier: func(c *Config) {
				c.MaxSlippagePercentage = decimal.Zero
			},
			expectedErr: nil,
		},
		{
			name: "Valid MaxSlippagePercentage (100)",
			modifier: func(c *Config) {
				c.MaxSlippagePercentage = decimal.NewFromInt(100)
			},
			expectedErr: nil,
		},
		{
			name: "Invalid MaxPositionPerUser (zero)",
			modifier: func(c *Config) {
				c.MaxPositionPerUser = decimal.Zero
			},
			expectedErr: models.ErrInvalidPositionLimit,
		},
		{
			name: "Invalid MaxPositionPerUser (negative)",
			modifier: func(c *Config) {
				c.MaxPositionPerUser = decimal.NewFromFloat(-100)
			},
			expectedErr: models.ErrInvalidPositionLimit,
		},
		{
			name: "Invalid BetTimeoutSeconds (zero)",
			modifier: func(c *Config) {
				c.BetTimeoutSeconds = 0
			},
			expectedErr: models.ErrInvalidBetTimeout,
		},
		{
			name: "Invalid BetTimeoutSeconds (negative)",
			modifier: func(c *Config) {
				c.BetTimeoutSeconds = -5
			},
			expectedErr: models.ErrInvalidBetTimeout,
		},
		{
			name: "Invalid BetTimeoutSeconds (too large)",
			modifier: func(c *Config) {
				c.BetTimeoutSeconds = 301
			},
			expectedErr: models.ErrInvalidBetTimeout,
		},
		{
			name: "Invalid MaxBetsPerMinute (zero)",
			modifier: func(c *Config) {
				c.MaxBetsPerMinute = 0
			},
			expectedErr: models.ErrInvalidRateLimit,
		},
		{
			name: "Invalid MaxBetsPerMinute (negative)",
			modifier: func(c *Config) {
				c.MaxBetsPerMinute = -1
			},
			expectedErr: models.ErrInvalidRateLimit,
		},
		{
			name: "Invalid MaxBetsPerMinute (too large)",
			modifier: func(c *Config) {
				c.MaxBetsPerMinute = 101
			},
			expectedErr: models.ErrInvalidRateLimit,
		},
		{
			name: "Invalid CooldownPeriod (negative)",
			modifier: func(c *Config) {
				c.CooldownPeriod = -1 * time.Second
			},
			expectedErr: models.ErrInvalidCooldownPeriod,
		},
		{
			name: "Valid CooldownPeriod (zero)",
			modifier: func(c *Config) {
				c.CooldownPeriod = 0
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := GetDefaultConfig()
			tt.modifier(config)
			err := config.Validate()

			if tt.expectedErr != nil {
				assert.Error(t, err, "Expected an error for test case: %s", tt.name)
				assert.Equal(t, tt.expectedErr, err, "Error mismatch for test case: %s", tt.name)
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s, but got: %v", tt.name, err)
			}
		})
	}
}
