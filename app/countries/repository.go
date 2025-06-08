package countries

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

// NewRepository creates a new country repository
func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

// GetAll returns all countries
func (r *repository) GetAll(ctx context.Context) ([]models.Country, error) {
	var countries []models.Country
	err := r.db.WithContext(ctx).Find(&countries).Error
	return countries, err
}

// GetByID returns a country by ID
func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*models.Country, error) {
	var country models.Country
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&country).Error
	if err != nil {
		return nil, err
	}
	return &country, nil
}

// GetByCode returns a country by code
func (r *repository) GetByCode(ctx context.Context, code string) (*models.Country, error) {
	var country models.Country
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&country).Error
	if err != nil {
		return nil, err
	}
	return &country, nil
}

// GetActive returns all active countries
func (r *repository) GetActive(ctx context.Context) ([]models.Country, error) {
	var countries []models.Country
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&countries).Error
	return countries, err
}

// Create creates a new country
func (r *repository) Create(ctx context.Context, country *models.Country) error {
	return r.db.WithContext(ctx).Create(country).Error
}

// Update updates an existing country
func (r *repository) Update(ctx context.Context, country *models.Country) error {
	return r.db.WithContext(ctx).Save(country).Error
}

// Delete deletes a country by ID
func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Country{}, id).Error
}
