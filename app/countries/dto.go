package countries

import (
	"time"

	"github.com/google/uuid"
	"github.com/joefazee/neo/models"
	"github.com/shopspring/decimal"
)

// CreateCountryRequest represents the request to create a country
type CreateCountryRequest struct {
	Name           string          `json:"name" binding:"required,min=2,max=100"`
	Code           string          `json:"code" binding:"required,len=3,uppercase"`
	CurrencyCode   string          `json:"currency_code" binding:"required,len=3,uppercase"`
	CurrencySymbol string          `json:"currency_symbol" binding:"required,min=1,max=10"`
	ContractUnit   decimal.Decimal `json:"contract_unit" binding:"required,gt=0"`
	MinBet         decimal.Decimal `json:"min_bet" binding:"required,gt=0"`
	MaxBet         decimal.Decimal `json:"max_bet" binding:"required,gt=0"`
	KYCRequired    bool            `json:"kyc_required"`
	Timezone       string          `json:"timezone,omitempty"`
}

// UpdateCountryRequest represents the request to update a country
type UpdateCountryRequest struct {
	Name           *string          `json:"name,omitempty" binding:"omitempty,min=2,max=100"`
	CurrencySymbol *string          `json:"currency_symbol,omitempty" binding:"omitempty,min=1,max=10"`
	IsActive       *bool            `json:"is_active,omitempty"`
	ContractUnit   *decimal.Decimal `json:"contract_unit,omitempty" binding:"omitempty,gt=0"`
	MinBet         *decimal.Decimal `json:"min_bet,omitempty" binding:"omitempty,gt=0"`
	MaxBet         *decimal.Decimal `json:"max_bet,omitempty" binding:"omitempty,gt=0"`
	KYCRequired    *bool            `json:"kyc_required,omitempty"`
	Timezone       *string          `json:"timezone,omitempty"`
}

// CountryResponse represents the response for country data
type CountryResponse struct {
	ID             uuid.UUID             `json:"id"`
	Name           string                `json:"name"`
	Code           string                `json:"code"`
	CurrencyCode   string                `json:"currency_code"`
	CurrencySymbol string                `json:"currency_symbol"`
	IsActive       bool                  `json:"is_active"`
	Config         CountryConfigResponse `json:"config"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

// CountryConfigResponse represents the country configuration
type CountryConfigResponse struct {
	ContractUnit decimal.Decimal `json:"contract_unit"`
	MinBet       decimal.Decimal `json:"min_bet"`
	MaxBet       decimal.Decimal `json:"max_bet"`
	KYCRequired  bool            `json:"kyc_required"`
	TaxRate      decimal.Decimal `json:"tax_rate,omitempty"`
	Timezone     string          `json:"timezone,omitempty"`
}

// ToCountryResponse converts a models.Country to CountryResponse
func ToCountryResponse(country *models.Country) *CountryResponse {
	res := &CountryResponse{
		ID:             country.ID,
		Name:           country.Name,
		Code:           country.Code,
		CurrencyCode:   country.CurrencyCode,
		CurrencySymbol: country.CurrencySymbol,
		IsActive:       *country.IsActive,
		CreatedAt:      country.CreatedAt,
		UpdatedAt:      country.UpdatedAt,
	}

	if country.Config != nil {
		res.Config = CountryConfigResponse{
			ContractUnit: country.Config.ContractUnit,
			MinBet:       country.Config.MinBet,
			MaxBet:       country.Config.MaxBet,
			KYCRequired:  country.Config.KYCRequired,
			TaxRate:      country.Config.TaxRate,
			Timezone:     country.Config.Timezone,
		}
	} else {
		res.Config = CountryConfigResponse{}
	}

	return res
}

// ToCountryResponseList converts a slice of models.Country to CountryResponse
func ToCountryResponseList(countries []models.Country) []CountryResponse {
	responses := make([]CountryResponse, len(countries))
	for i := range countries {
		responses[i] = *ToCountryResponse(&countries[i])
	}
	return responses
}
