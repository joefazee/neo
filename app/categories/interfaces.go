package categories

import (
	"context"

	"github.com/google/uuid"
	"github.com/joefazee/neo/models"
)

// Repository defines the interface for category data access
type Repository interface {
	GetByCountryID(ctx context.Context, countryID uuid.UUID) ([]models.Category, error)
	GetActiveByCountryID(ctx context.Context, countryID uuid.UUID) ([]models.Category, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Category, error)
	GetBySlug(ctx context.Context, countryID uuid.UUID, slug string) (*models.Category, error)
	Create(ctx context.Context, category *models.Category) error
	Update(ctx context.Context, category *models.Category) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// Service defines the interface for category business logic
type Service interface {
	GetCategoriesByCountry(ctx context.Context, countryID uuid.UUID) ([]CategoryResponse, error)
	GetActiveCategoriesByCountry(ctx context.Context, countryID uuid.UUID) ([]CategoryResponse, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*CategoryResponse, error)
	GetCategoryBySlug(ctx context.Context, countryID uuid.UUID, slug string) (*CategoryResponse, error)
	CreateCategory(ctx context.Context, req CreateCategoryRequest) (*CategoryResponse, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, req UpdateCategoryRequest) (*CategoryResponse, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
}
