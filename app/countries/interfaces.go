package countries

import (
	"context"

	"github.com/google/uuid"
	"github.com/joefazee/neo/models"
)

// Repository defines the interface for country data access
type Repository interface {
	GetAll(ctx context.Context) ([]models.Country, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Country, error)
	GetByCode(ctx context.Context, code string) (*models.Country, error)
	GetActive(ctx context.Context) ([]models.Country, error)
	Create(ctx context.Context, country *models.Country) error
	Update(ctx context.Context, country *models.Country) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// Service defines the interface for country business logic
type Service interface {
	GetAllCountries(ctx context.Context) ([]CountryResponse, error)
	GetCountryByID(ctx context.Context, id uuid.UUID) (*CountryResponse, error)
	GetCountryByCode(ctx context.Context, code string) (*CountryResponse, error)
	GetActiveCountries(ctx context.Context) ([]CountryResponse, error)
	CreateCountry(ctx context.Context, req *CreateCountryRequest) (*CountryResponse, error)
	UpdateCountry(ctx context.Context, id uuid.UUID, req UpdateCountryRequest) (*CountryResponse, error)
	DeleteCountry(ctx context.Context, id uuid.UUID) error
}
