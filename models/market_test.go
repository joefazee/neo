package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestSafeguardConfig(t *testing.T) {
	t.Run("Value and Scan", func(t *testing.T) {
		config := SafeguardConfig{
			MinQuorumAmount:    decimal.NewFromFloat(1000),
			MinOutcomes:        2,
			HouseBotEnabled:    true,
			HouseBotAmount:     decimal.NewFromFloat(500),
			ImbalanceThreshold: decimal.NewFromFloat(0.8),
			VoidOnQuorumFail:   true,
		}

		value, err := config.Value()
		assert.NoError(t, err)

		var result SafeguardConfig
		err = result.Scan(value)
		assert.NoError(t, err)
		assert.True(t, config.MinQuorumAmount.Equal(result.MinQuorumAmount))
		assert.Equal(t, config.MinOutcomes, result.MinOutcomes)
		assert.Equal(t, config.HouseBotEnabled, result.HouseBotEnabled)

		err = result.Scan(nil)
		assert.NoError(t, err)

		cfgBS, err := json.Marshal(config)
		assert.NoError(t, err)

		err = result.Scan(string(cfgBS))
		assert.NoError(t, err)

		err = result.Scan(func() {})
		assert.NoError(t, err)
	})
}

func TestOracleConfig(t *testing.T) {
	t.Run("Value and Scan", func(t *testing.T) {
		config := OracleConfig{
			Provider:       "chainlink",
			DataSource:     "api.example.com",
			ResolutionURL:  "https://example.com/resolve",
			Criteria:       map[string]string{"type": "price"},
			AutoResolve:    true,
			BackupProvider: "manual",
		}

		value, err := config.Value()
		assert.NoError(t, err)

		var result OracleConfig
		err = result.Scan(value)
		assert.NoError(t, err)
		assert.Equal(t, config.Provider, result.Provider)
		assert.Equal(t, config.AutoResolve, result.AutoResolve)
		assert.Equal(t, config.Criteria["type"], result.Criteria["type"])

		cfgBS, err := json.Marshal(config)
		assert.NoError(t, err)
		err = result.Scan(string(cfgBS))
		assert.NoError(t, err)

		err = result.Scan(func() {})
		assert.NoError(t, err)

		err = result.Scan(nil)
		assert.NoError(t, err)
	})
}

func TestMarketMetadata(t *testing.T) {
	t.Run("Value and Scan", func(t *testing.T) {
		futureTime := time.Now().Add(24 * time.Hour)
		metadata := MarketMetadata{
			Tags:          []string{"sports", "football"},
			ImageURL:      "https://example.com/image.png",
			SourceURL:     "https://example.com/source",
			FeaturedUntil: &futureTime,
			ViewCount:     1500,
		}

		value, err := metadata.Value()
		assert.NoError(t, err)

		var result MarketMetadata
		err = result.Scan(value)
		assert.NoError(t, err)
		assert.Equal(t, metadata.Tags, result.Tags)
		assert.Equal(t, metadata.ImageURL, result.ImageURL)
		assert.Equal(t, metadata.ViewCount, result.ViewCount)

		cfgBS, err := json.Marshal(metadata)
		assert.NoError(t, err)

		err = result.Scan(string(cfgBS))
		assert.NoError(t, err)

		err = result.Scan(func() {})
		assert.NoError(t, err)

		err = result.Scan(nil)
		assert.NoError(t, err)
	})
}

func TestMarket(t *testing.T) {
	t.Run("TableName", func(t *testing.T) {
		m := Market{}
		assert.Equal(t, "markets", m.TableName())
	})

	t.Run("BeforeCreate", func(t *testing.T) {
		m := Market{}
		assert.Equal(t, uuid.Nil, m.ID)

		err := m.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, m.ID)

		existingID := uuid.New()
		m2 := Market{ID: existingID}
		err = m2.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, existingID, m2.ID)
	})

	t.Run("Status checks", func(t *testing.T) {
		futureTime := time.Now().Add(time.Hour)
		pastTime := time.Now().Add(-time.Hour)

		m := Market{
			Status:    MarketStatusOpen,
			CloseTime: futureTime,
		}
		assert.True(t, m.IsOpen())
		assert.False(t, m.IsClosed())
		assert.True(t, m.CanBet())

		m.CloseTime = pastTime
		assert.False(t, m.IsOpen())
		assert.True(t, m.IsClosed())
		assert.False(t, m.CanBet())

		m.Status = MarketStatusResolved
		resolvedAt := time.Now()
		m.ResolvedAt = &resolvedAt
		assert.True(t, m.IsResolved())
		assert.False(t, m.CanResolve())

		m.Status = MarketStatusVoided
		assert.True(t, m.IsVoided())
		assert.False(t, m.CanBet())

		m.Status = MarketStatusClosed
		m.ResolvedAt = nil
		assert.True(t, m.CanResolve())
	})

	t.Run("Financial calculations", func(t *testing.T) {
		m := Market{
			RakePercentage:      decimal.NewFromFloat(0.05),
			CreatorRevenueShare: decimal.NewFromFloat(0.5),
		}

		betAmount := decimal.NewFromFloat(100)
		rakeAmount := m.GetRakeAmount(betAmount)
		expectedRake := decimal.NewFromFloat(5)
		assert.True(t, expectedRake.Equal(rakeAmount))

		creatorFee := m.GetCreatorFee(rakeAmount)
		expectedFee := decimal.NewFromFloat(2.5)
		assert.True(t, expectedFee.Equal(creatorFee))
	})

	t.Run("HasMinQuorum", func(t *testing.T) {
		m := Market{
			TotalPoolAmount: decimal.NewFromFloat(1500),
			SafeguardConfig: SafeguardConfig{
				MinQuorumAmount: decimal.NewFromFloat(1000),
			},
		}
		assert.True(t, m.HasMinQuorum())

		m.TotalPoolAmount = decimal.NewFromFloat(500)
		assert.False(t, m.HasMinQuorum())

		m.SafeguardConfig.MinQuorumAmount = decimal.Zero
		assert.True(t, m.HasMinQuorum())
	})

	t.Run("ValidateBetAmount", func(t *testing.T) {
		maxBet := decimal.NewFromFloat(1000)
		m := Market{
			MinBetAmount: decimal.NewFromFloat(10),
			MaxBetAmount: &maxBet,
		}

		err := m.ValidateBetAmount(decimal.NewFromFloat(50))
		assert.NoError(t, err)

		err = m.ValidateBetAmount(decimal.NewFromFloat(5))
		assert.Equal(t, ErrBetTooSmall, err)

		err = m.ValidateBetAmount(decimal.NewFromFloat(1500))
		assert.Equal(t, ErrBetTooLarge, err)

		m.MaxBetAmount = nil
		err = m.ValidateBetAmount(decimal.NewFromFloat(1500))
		assert.NoError(t, err)
	})

	t.Run("Resolve", func(t *testing.T) {
		m := Market{
			Status:    MarketStatusClosed,
			CloseTime: time.Now().Add(-time.Hour),
		}

		err := m.Resolve("outcome_a", "manual resolution")
		assert.NoError(t, err)
		assert.Equal(t, MarketStatusResolved, m.Status)
		assert.Equal(t, "outcome_a", m.ResolvedOutcome)
		assert.Equal(t, "manual resolution", m.ResolutionSource)
		assert.NotNil(t, m.ResolvedAt)

		err = m.Resolve("outcome_b", "another source")
		assert.Equal(t, ErrMarketNotOpen, err)
	})

	t.Run("Void", func(t *testing.T) {
		m := Market{Status: MarketStatusOpen}

		err := m.Void()
		assert.NoError(t, err)
		assert.Equal(t, MarketStatusVoided, m.Status)

		m.Status = MarketStatusResolved
		now := time.Now()
		m.ResolvedAt = &now
		err = m.Void()
		assert.Equal(t, ErrMarketAlreadyClosed, err)
	})

	t.Run("Validate", func(t *testing.T) {
		futureClose := time.Now().Add(time.Hour)
		futureResolution := time.Now().Add(2 * time.Hour)

		validMarket := Market{
			CountryID:          uuid.New(),
			CategoryID:         uuid.New(),
			Title:              "Test Market",
			CloseTime:          futureClose,
			ResolutionDeadline: futureResolution,
			MinBetAmount:       decimal.NewFromFloat(10),
		}
		assert.NoError(t, validMarket.Validate())

		tests := []struct {
			name   string
			modify func(*Market)
			err    error
		}{
			{"Invalid CountryID", func(m *Market) { m.CountryID = uuid.Nil }, ErrInvalidCountryID},
			{"Invalid CategoryID", func(m *Market) { m.CategoryID = uuid.Nil }, ErrInvalidCategoryName},
			{"Empty Title", func(m *Market) { m.Title = "" }, ErrInvalidMarketTitle},
			{"Past CloseTime", func(m *Market) { m.CloseTime = time.Now().Add(-time.Hour) }, ErrInvalidCloseTime},
			{"Invalid Resolution", func(m *Market) { m.ResolutionDeadline = m.CloseTime.Add(-time.Hour) }, ErrInvalidResolutionTime},
			{"Invalid MinBet", func(m *Market) { m.MinBetAmount = decimal.Zero }, ErrInvalidBetAmount},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				market := validMarket
				tt.modify(&market)
				assert.Equal(t, tt.err, market.Validate())
			})
		}
	})
}
