package categories

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/joefazee/neo/models"
)

// repository implements the Repository interface using GORM
type repository struct {
	db *gorm.DB
}

// NewRepository creates a new category repository
func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

// GetByCountryID returns all categories for a country
func (r *repository) GetByCountryID(ctx context.Context, countryID uuid.UUID) ([]models.Category, error) {
	var categories []models.Category
	err := r.db.WithContext(ctx).
		Where("country_id = ?", countryID).
		Order("sort_order ASC, name ASC").
		Find(&categories).Error
	return categories, err
}

// GetActiveByCountryID returns all active categories for a country
func (r *repository) GetActiveByCountryID(ctx context.Context, countryID uuid.UUID) ([]models.Category, error) {
	var categories []models.Category
	err := r.db.WithContext(ctx).
		Where("country_id = ? AND is_active = ?", countryID, true).
		Order("sort_order ASC, name ASC").
		Find(&categories).Error
	return categories, err
}

// GetByID returns a category by ID
func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*models.Category, error) {
	var category models.Category
	err := r.db.WithContext(ctx).
		Preload("Country").
		Where("id = ?", id).
		First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// GetBySlug returns a category by slug within a country
func (r *repository) GetBySlug(ctx context.Context, countryID uuid.UUID, slug string) (*models.Category, error) {
	var category models.Category
	err := r.db.WithContext(ctx).
		Preload("Country").
		Where("country_id = ? AND slug = ?", countryID, slug).
		First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// Create creates a new category
func (r *repository) Create(ctx context.Context, category *models.Category) error {
	return r.db.WithContext(ctx).Create(category).Error
}

// Update updates an existing category
func (r *repository) Update(ctx context.Context, category *models.Category) error {
	return r.db.WithContext(ctx).Save(category).Error
}

// Delete deletes a category by ID
func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Category{}, id).Error
}
