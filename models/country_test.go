package models

import (
	"database/sql/driver"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func newEmptyCountry() *Country {
	return &Country{
		Config: &CountryConfig{},
	}
}
func TestCountryConfig_Value(t *testing.T) {
	tests := []struct {
		name     string
		config   CountryConfig
		wantErr  bool
		validate func(t *testing.T, result driver.Value)
	}{
		{
			name: "valid config with all fields",
			config: CountryConfig{
				ContractUnit: decimal.NewFromFloat(1.5),
				MinBet:       decimal.NewFromFloat(10.0),
				MaxBet:       decimal.NewFromFloat(1000.0),
				KYCRequired:  true,
				TaxRate:      decimal.NewFromFloat(0.15),
				Timezone:     "UTC",
			},
			wantErr: false,
			validate: func(t *testing.T, result driver.Value) {
				bytes, ok := result.([]byte)
				assert.True(t, ok, "result should be []byte")

				var unmarshaled CountryConfig
				err := json.Unmarshal(bytes, &unmarshaled)
				assert.NoError(t, err)
				assert.True(t, unmarshaled.ContractUnit.Equal(decimal.NewFromFloat(1.5)))
				assert.True(t, unmarshaled.MinBet.Equal(decimal.NewFromFloat(10.0)))
				assert.True(t, unmarshaled.MaxBet.Equal(decimal.NewFromFloat(1000.0)))
				assert.True(t, unmarshaled.KYCRequired)
				assert.True(t, unmarshaled.TaxRate.Equal(decimal.NewFromFloat(0.15)))
				assert.Equal(t, "UTC", unmarshaled.Timezone)
			},
		},
		{
			name: "config with only required fields",
			config: CountryConfig{
				ContractUnit: decimal.NewFromFloat(1.0),
				MinBet:       decimal.NewFromFloat(5.0),
				MaxBet:       decimal.NewFromFloat(500.0),
				KYCRequired:  false,
			},
			wantErr: false,
			validate: func(t *testing.T, result driver.Value) {
				bytes, ok := result.([]byte)
				assert.True(t, ok, "result should be []byte")

				var unmarshaled CountryConfig
				err := json.Unmarshal(bytes, &unmarshaled)
				assert.NoError(t, err)
				assert.True(t, unmarshaled.ContractUnit.Equal(decimal.NewFromFloat(1.0)))
				assert.False(t, unmarshaled.KYCRequired)
				assert.True(t, unmarshaled.TaxRate.IsZero())
				assert.Empty(t, unmarshaled.Timezone)
			},
		},
		{
			name: "zero values config",
			config: CountryConfig{
				ContractUnit: decimal.Zero,
				MinBet:       decimal.Zero,
				MaxBet:       decimal.Zero,
				KYCRequired:  false,
			},
			wantErr: false,
			validate: func(t *testing.T, result driver.Value) {
				bytes, ok := result.([]byte)
				assert.True(t, ok, "result should be []byte")
				assert.Contains(t, string(bytes), "contract_unit")
			},
		},
		{
			name: "config with large decimal values",
			config: CountryConfig{
				ContractUnit: decimalFromString("999999999.999999999"),
				MinBet:       decimalFromString("0.000000001"),
				MaxBet:       decimalFromString("999999999999.99"),
				KYCRequired:  true,
				TaxRate:      decimalFromString("0.999"),
			},
			wantErr: false,
			validate: func(t *testing.T, result driver.Value) {
				bytes, ok := result.([]byte)
				assert.True(t, ok, "result should be []byte")

				var unmarshaled CountryConfig
				err := json.Unmarshal(bytes, &unmarshaled)
				assert.NoError(t, err)
				expected, _ := decimal.NewFromString("999999999.999999999")
				assert.True(t, unmarshaled.ContractUnit.Equal(expected))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.config.Value()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestCountryConfig_Scan(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		wantErr  bool
		validate func(t *testing.T, config *CountryConfig)
	}{
		{
			name:    "nil value",
			input:   nil,
			wantErr: false,
			validate: func(t *testing.T, config *CountryConfig) {
				// Should remain unchanged from zero value
				assert.True(t, config.ContractUnit.IsZero())
				assert.True(t, config.MinBet.IsZero())
				assert.True(t, config.MaxBet.IsZero())
				assert.False(t, config.KYCRequired)
			},
		},
		{
			name: "valid JSON as []byte",
			input: []byte(`{
				"contract_unit": "1.5",
				"min_bet": "10.0",
				"max_bet": "1000.0",
				"kyc_required": true,
				"tax_rate": "0.15",
				"timezone": "America/New_York"
			}`),
			wantErr: false,
			validate: func(t *testing.T, config *CountryConfig) {
				assert.True(t, config.ContractUnit.Equal(decimal.NewFromFloat(1.5)))
				assert.True(t, config.MinBet.Equal(decimal.NewFromFloat(10.0)))
				assert.True(t, config.MaxBet.Equal(decimal.NewFromFloat(1000.0)))
				assert.True(t, config.KYCRequired)
				assert.True(t, config.TaxRate.Equal(decimal.NewFromFloat(0.15)))
				assert.Equal(t, "America/New_York", config.Timezone)
			},
		},
		{
			name: "valid JSON as string",
			input: `{
				"contract_unit": "2.0",
				"min_bet": "5.0",
				"max_bet": "500.0",
				"kyc_required": false
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *CountryConfig) {
				assert.True(t, config.ContractUnit.Equal(decimal.NewFromFloat(2.0)))
				assert.True(t, config.MinBet.Equal(decimal.NewFromFloat(5.0)))
				assert.True(t, config.MaxBet.Equal(decimal.NewFromFloat(500.0)))
				assert.False(t, config.KYCRequired)
				assert.True(t, config.TaxRate.IsZero())
				assert.Empty(t, config.Timezone)
			},
		},
		{
			name:     "invalid JSON as []byte",
			input:    []byte(`{"contract_unit": "invalid", "min_bet":`),
			wantErr:  true,
			validate: nil,
		},
		{
			name:     "invalid JSON as string",
			input:    `{"contract_unit": "1.0", "min_bet": }`,
			wantErr:  true,
			validate: nil,
		},
		{
			name:     "empty []byte",
			input:    []byte(``),
			wantErr:  true,
			validate: nil,
		},
		{
			name:     "empty string",
			input:    "",
			wantErr:  true,
			validate: nil,
		},
		{
			name: "JSON with extra fields (should be ignored)",
			input: []byte(`{
				"contract_unit": "1.0",
				"min_bet": "5.0",
				"max_bet": "100.0",
				"kyc_required": true,
				"unknown_field": "should_be_ignored",
				"another_unknown": 123
			}`),
			wantErr: false,
			validate: func(t *testing.T, config *CountryConfig) {
				assert.True(t, config.ContractUnit.Equal(decimal.NewFromFloat(1.0)))
				assert.True(t, config.MinBet.Equal(decimal.NewFromFloat(5.0)))
				assert.True(t, config.MaxBet.Equal(decimal.NewFromFloat(100.0)))
				assert.True(t, config.KYCRequired)
			},
		},
		{
			name:    "unsupported type (int)",
			input:   123,
			wantErr: false,
			validate: func(t *testing.T, config *CountryConfig) {
				assert.True(t, config.ContractUnit.IsZero())
			},
		},
		{
			name:    "unsupported type (bool)",
			input:   true,
			wantErr: false,
			validate: func(t *testing.T, config *CountryConfig) {
				assert.True(t, config.ContractUnit.IsZero())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &CountryConfig{}
			err := config.Scan(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

func TestCountryConfig_RoundTrip(t *testing.T) {
	original := CountryConfig{
		ContractUnit: decimal.NewFromFloat(1.5),
		MinBet:       decimal.NewFromFloat(10.0),
		MaxBet:       decimal.NewFromFloat(1000.0),
		KYCRequired:  true,
		TaxRate:      decimal.NewFromFloat(0.15),
		Timezone:     "Europe/London",
	}

	value, err := original.Value()
	assert.NoError(t, err)
	assert.NotNil(t, value)

	var scanned CountryConfig
	err = scanned.Scan(value)
	assert.NoError(t, err)

	assert.True(t, scanned.ContractUnit.Equal(original.ContractUnit))
	assert.True(t, scanned.MinBet.Equal(original.MinBet))
	assert.True(t, scanned.MaxBet.Equal(original.MaxBet))
	assert.Equal(t, original.KYCRequired, scanned.KYCRequired)
	assert.True(t, scanned.TaxRate.Equal(original.TaxRate))
	assert.Equal(t, original.Timezone, scanned.Timezone)
}

func TestCountryConfig_EdgeCases(t *testing.T) {
	t.Run("scan with null bytes", func(t *testing.T) {
		config := &CountryConfig{}
		err := config.Scan([]byte("null"))
		assert.NoError(t, err)
		assert.True(t, config.ContractUnit.IsZero())
	})

	t.Run("scan with malformed decimal in JSON", func(t *testing.T) {
		config := &CountryConfig{}
		input := []byte(`{"contract_unit": "not-a-number", "min_bet": "10.0", "max_bet": "100.0", "kyc_required": false}`)
		err := config.Scan(input)
		assert.Error(t, err)
	})

	t.Run("value with very precise decimals", func(t *testing.T) {
		config := CountryConfig{
			ContractUnit: decimalFromString("0.123456789012345678"),
			MinBet:       decimalFromString("1.000000000000000001"),
			MaxBet:       decimalFromString("999999999999999999.999999999999999999"),
			KYCRequired:  true,
		}

		value, err := config.Value()
		assert.NoError(t, err)

		var scanned CountryConfig
		err = scanned.Scan(value)
		assert.NoError(t, err)

		expected, _ := decimal.NewFromString("0.123456789012345678")
		assert.True(t, scanned.ContractUnit.Equal(expected))
	})
}

func decimalFromString(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

func TestCountry(t *testing.T) {
	t.Run("TableName", func(t *testing.T) {
		c := Country{}
		assert.Equal(t, "countries", c.TableName())
	})

	t.Run("BeforeCreate", func(t *testing.T) {
		c := Country{}
		assert.Equal(t, uuid.Nil, c.ID)

		err := c.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, c.ID)

		existingID := uuid.New()
		c2 := Country{ID: existingID}
		err = c2.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, existingID, c2.ID)
	})

	t.Run("IsValidCurrency", func(t *testing.T) {
		tests := []struct {
			name         string
			currencyCode string
			expected     bool
		}{
			{"Valid currency", "NGN", true},
			{"Valid currency USD", "USD", true},
			{"Empty currency", "", false},
			{"Too short", "NG", false},
			{"Too long", "NGNN", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := Country{CurrencyCode: tt.currencyCode}
				assert.Equal(t, tt.expected, c.IsValidCurrency())
			})
		}
	})

	t.Run("GetMinBetAmount", func(t *testing.T) {
		c := Country{
			Config: &CountryConfig{
				MinBet: decimal.NewFromInt(100),
			},
		}
		assert.Equal(t, decimal.NewFromInt(100), c.GetMinBetAmount())

		c.Config = &CountryConfig{MinBet: decimal.NewFromFloat(200.0)}
		assert.Equal(t, decimal.NewFromFloat(200.0), c.GetMinBetAmount())
	})

	t.Run("GetMaxBetAmount", func(t *testing.T) {
		c := newEmptyCountry()
		assert.Equal(t, decimal.NewFromInt(50000), c.GetMaxBetAmount())

		c.Config = &CountryConfig{MaxBet: decimal.NewFromFloat(10000.0)}
		assert.Equal(t, decimal.NewFromFloat(10000.0), c.GetMaxBetAmount())
	})

	t.Run("GetContractUnit", func(t *testing.T) {
		c := newEmptyCountry()
		assert.Equal(t, decimal.NewFromInt(100), c.GetContractUnit())

		c.Config = &CountryConfig{ContractUnit: decimal.NewFromFloat(50.0)}
		assert.Equal(t, decimal.NewFromFloat(50.0), c.GetContractUnit())
	})

	t.Run("RequiresKYC", func(t *testing.T) {
		c := newEmptyCountry()
		assert.False(t, c.RequiresKYC())

		c.Config = &CountryConfig{KYCRequired: true}
		assert.True(t, c.RequiresKYC())

		c.Config = &CountryConfig{KYCRequired: false}
		assert.False(t, c.RequiresKYC())
	})

	t.Run("Validate", func(t *testing.T) {
		tests := []struct {
			name        string
			country     Country
			expectedErr error
		}{
			{
				name: "Valid country",
				country: Country{
					Name:           "Nigeria",
					Code:           "NGN",
					CurrencyCode:   "NGN",
					CurrencySymbol: "₦",
				},
				expectedErr: nil,
			},
			{
				name: "Empty name",
				country: Country{
					Name:           "",
					Code:           "NGN",
					CurrencyCode:   "NGN",
					CurrencySymbol: "₦",
				},
				expectedErr: ErrInvalidCountryName,
			},
			{
				name: "Invalid code length",
				country: Country{
					Name:           "Nigeria",
					Code:           "NG",
					CurrencyCode:   "NGN",
					CurrencySymbol: "₦",
				},
				expectedErr: ErrInvalidCountryCode,
			},
			{
				name: "Invalid currency code",
				country: Country{
					Name:           "Nigeria",
					Code:           "NGN",
					CurrencyCode:   "NG",
					CurrencySymbol: "₦",
				},
				expectedErr: ErrInvalidCurrencyCode,
			},
			{
				name: "Empty currency symbol",
				country: Country{
					Name:           "Nigeria",
					Code:           "NGN",
					CurrencyCode:   "NGN",
					CurrencySymbol: "",
				},
				expectedErr: ErrInvalidCurrencySymbol,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.country.Validate()
				if tt.expectedErr != nil {
					assert.Equal(t, tt.expectedErr, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("Full integration", func(t *testing.T) {
		isActive := true
		c := Country{
			ID:             uuid.New(),
			Name:           "Nigeria",
			Code:           "NGN",
			CurrencyCode:   "NGN",
			CurrencySymbol: "₦",
			IsActive:       &isActive,
			Config: &CountryConfig{
				MinBet:       decimal.NewFromFloat(200.0),
				MaxBet:       decimal.NewFromFloat(10000.0),
				ContractUnit: decimal.NewFromFloat(50.0),
				KYCRequired:  true,
			},
		}

		assert.NotEmpty(t, c.ID)
		assert.True(t, c.IsValidCurrency())
		assert.Equal(t, decimal.NewFromFloat(200.0), c.GetMinBetAmount())
		assert.Equal(t, decimal.NewFromFloat(10000.0), c.GetMaxBetAmount())
		assert.Equal(t, decimal.NewFromFloat(50.0), c.GetContractUnit())
		assert.True(t, c.RequiresKYC())
		assert.NoError(t, c.Validate())
	})
}
