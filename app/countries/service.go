package countries

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/joefazee/neo/models"
)

// service implements the Service interface
type service struct {
	repo Repository
}

// NewService creates a new country service
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

// GetAllCountries returns all countries
func (s *service) GetAllCountries(ctx context.Context) ([]CountryResponse, error) {
	countries, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return ToCountryResponseList(countries), nil
}

// GetCountryByID returns a country by ID
func (s *service) GetCountryByID(ctx context.Context, id uuid.UUID) (*CountryResponse, error) {
	country, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, err
	}
	return ToCountryResponse(country), nil
}

// GetCountryByCode returns a country by code
func (s *service) GetCountryByCode(ctx context.Context, code string) (*CountryResponse, error) {
	country, err := s.repo.GetByCode(ctx, strings.ToUpper(code))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, err
	}
	return ToCountryResponse(country), nil
}

// GetActiveCountries returns all active countries
func (s *service) GetActiveCountries(ctx context.Context) ([]CountryResponse, error) {
	countries, err := s.repo.GetActive(ctx)
	if err != nil {
		return nil, err
	}
	return ToCountryResponseList(countries), nil
}

// CreateCountry creates a new country
func (s *service) CreateCountry(ctx context.Context, req *CreateCountryRequest) (*CountryResponse, error) {
	// Validate min/max bet amounts
	if req.MinBet.GreaterThanOrEqual(req.MaxBet) {
		return nil, errors.New("min_bet must be less than max_bet")
	}

	// Check if country code already exists
	existing, err := s.repo.GetByCode(ctx, req.Code)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("country with this code already exists")
	}
	isActive := true
	country := &models.Country{
		Name:           req.Name,
		Code:           strings.ToUpper(req.Code),
		CurrencyCode:   strings.ToUpper(req.CurrencyCode),
		CurrencySymbol: req.CurrencySymbol,
		IsActive:       &isActive,
		Config: &models.CountryConfig{
			ContractUnit: req.ContractUnit,
			MinBet:       req.MinBet,
			MaxBet:       req.MaxBet,
			KYCRequired:  req.KYCRequired,
			Timezone:     req.Timezone,
		},
	}

	if err := country.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, country); err != nil {
		return nil, err
	}

	return ToCountryResponse(country), nil
}

// UpdateCountry updates an existing country
func (s *service) UpdateCountry(ctx context.Context, id uuid.UUID, req UpdateCountryRequest) (*CountryResponse, error) {
	country, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		country.Name = *req.Name
	}
	if req.CurrencySymbol != nil {
		country.CurrencySymbol = *req.CurrencySymbol
	}
	if req.IsActive != nil {
		country.IsActive = req.IsActive
	}

	// Update config fields
	if req.ContractUnit != nil {
		country.Config.ContractUnit = *req.ContractUnit
	}
	if req.MinBet != nil {
		country.Config.MinBet = *req.MinBet
	}
	if req.MaxBet != nil {
		country.Config.MaxBet = *req.MaxBet
	}
	if req.KYCRequired != nil {
		country.Config.KYCRequired = *req.KYCRequired
	}
	if req.Timezone != nil {
		country.Config.Timezone = *req.Timezone
	}

	// Validate min/max bet amounts if both are set
	if !country.Config.MinBet.IsZero() && !country.Config.MaxBet.IsZero() {
		if country.Config.MinBet.GreaterThanOrEqual(country.Config.MaxBet) {
			return nil, errors.New("min_bet must be less than max_bet")
		}
	}

	if err := country.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, country); err != nil {
		return nil, err
	}

	return ToCountryResponse(country), nil
}

// DeleteCountry deletes a country
func (s *service) DeleteCountry(ctx context.Context, id uuid.UUID) error {
	// Check if country exists
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ErrRecordNotFound
		}
		return err
	}

	return s.repo.Delete(ctx, id)
}
