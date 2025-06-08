package markets

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/joefazee/neo/models"
)

// repository implements the Repository interface using GORM
type repository struct {
	db *gorm.DB
}

// NewRepository creates a new market repository
func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

// GetAll returns markets with filters and pagination
func (r *repository) GetAll(ctx context.Context, filters *MarketFilters) ([]models.Market, int64, error) {
	var markets []models.Market
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Market{})

	// Apply filters
	query = r.applyFilters(query, filters)

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and sorting
	query = r.applySorting(query, filters)
	query = r.applyPagination(query, filters)

	// Preload related data
	query = query.Preload("Outcomes").Preload("Country").Preload("Category").Preload("Creator")

	err := query.Find(&markets).Error
	return markets, total, err
}

// GetByID returns a market by ID with all related data
func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*models.Market, error) {
	var market models.Market
	err := r.db.WithContext(ctx).
		Preload("Outcomes").
		Preload("Country").
		Preload("Category").
		Preload("Creator").
		Where("id = ?", id).
		First(&market).Error
	if err != nil {
		return nil, err
	}
	return &market, nil
}

// GetByStatus returns markets by status
func (r *repository) GetByStatus(ctx context.Context, status models.MarketStatus) ([]models.Market, error) {
	var markets []models.Market
	err := r.db.WithContext(ctx).
		Preload("Outcomes").
		Where("status = ?", status).
		Find(&markets).Error
	return markets, err
}

// GetByCountryAndCategory returns markets by country and category
func (r *repository) GetByCountryAndCategory(ctx context.Context, countryID, categoryID uuid.UUID) ([]models.Market, error) {
	var markets []models.Market
	err := r.db.WithContext(ctx).
		Preload("Outcomes").
		Where("country_id = ? AND category_id = ?", countryID, categoryID).
		Find(&markets).Error
	return markets, err
}

// GetByCreator returns markets created by a specific user
func (r *repository) GetByCreator(ctx context.Context, creatorID uuid.UUID) ([]models.Market, error) {
	var markets []models.Market
	err := r.db.WithContext(ctx).
		Preload("Outcomes").
		Where("creator_id = ?", creatorID).
		Order("created_at DESC").
		Find(&markets).Error
	return markets, err
}

// GetExpiredMarkets returns markets that have passed their close time but are not closed
func (r *repository) GetExpiredMarkets(ctx context.Context) ([]models.Market, error) {
	var markets []models.Market
	err := r.db.WithContext(ctx).
		Preload("Outcomes").
		Where("close_time < ? AND status = ?", time.Now(), models.MarketStatusOpen).
		Find(&markets).Error
	return markets, err
}

// Create creates a new market
func (r *repository) Create(ctx context.Context, market *models.Market) error {
	return r.db.WithContext(ctx).Create(market).Error
}

// Update updates an existing market
func (r *repository) Update(ctx context.Context, market *models.Market) error {
	return r.db.WithContext(ctx).Save(market).Error
}

// Delete deletes a market by ID
func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Market{}, id).Error
}

// GetMarketOutcomes returns all outcomes for a market
func (r *repository) GetMarketOutcomes(ctx context.Context, marketID uuid.UUID) ([]models.MarketOutcome, error) {
	var outcomes []models.MarketOutcome
	err := r.db.WithContext(ctx).
		Where("market_id = ?", marketID).
		Order("sort_order ASC, outcome_label ASC").
		Find(&outcomes).Error
	return outcomes, err
}

// CreateMarketOutcome creates a new market outcome
func (r *repository) CreateMarketOutcome(ctx context.Context, outcome *models.MarketOutcome) error {
	return r.db.WithContext(ctx).Create(outcome).Error
}

// UpdateMarketOutcome updates an existing market outcome
func (r *repository) UpdateMarketOutcome(ctx context.Context, outcome *models.MarketOutcome) error {
	return r.db.WithContext(ctx).Save(outcome).Error
}

// DeleteMarketOutcome deletes a market outcome by ID
func (r *repository) DeleteMarketOutcome(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.MarketOutcome{}, id).Error
}

// applyFilters applies search and filter criteria to the query
func (r *repository) applyFilters(query *gorm.DB, filters *MarketFilters) *gorm.DB {
	if filters == nil {
		return query
	}

	if filters.CountryID != nil {
		query = query.Where("country_id = ?", *filters.CountryID)
	}

	if filters.CategoryID != nil {
		query = query.Where("category_id = ?", *filters.CategoryID)
	}

	if filters.CreatorID != nil {
		query = query.Where("creator_id = ?", *filters.CreatorID)
	}

	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}

	if filters.MarketType != nil {
		query = query.Where("market_type = ?", *filters.MarketType)
	}

	if filters.Search != "" {
		searchTerm := "%" + strings.ToLower(filters.Search) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ?", searchTerm, searchTerm)
	}

	if len(filters.Tags) > 0 {
		for _, tag := range filters.Tags {
			query = query.Where("metadata->>'tags' LIKE ?", "%"+tag+"%")
		}
	}

	return query
}

// applySorting applies sorting to the query
func (r *repository) applySorting(query *gorm.DB, filters *MarketFilters) *gorm.DB {
	sortBy := filters.SortBy
	if sortBy == "" {
		sortBy = "created_at" // Default sort
	}

	sortOrder := filters.SortOrder
	if sortOrder == "" {
		sortOrder = "desc" // Default order
	}

	// Validate sort fields to prevent SQL injection
	validSortFields := map[string]bool{
		"created_at":        true,
		"updated_at":        true,
		"close_time":        true,
		"total_pool_amount": true,
		"title":             true,
	}

	if !validSortFields[sortBy] {
		sortBy = "created_at"
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	orderClause := fmt.Sprintf("%s %s", sortBy, sortOrder)
	return query.Order(orderClause)
}

// applyPagination applies pagination to the query
func (r *repository) applyPagination(query *gorm.DB, filters *MarketFilters) *gorm.DB {
	page := filters.Page
	if page < 1 {
		page = 1
	}

	perPage := filters.PerPage
	if perPage < 1 || perPage > 100 { // Limit max per page
		perPage = 20 // Default per page
	}

	offset := (page - 1) * perPage
	return query.Offset(offset).Limit(perPage)
}
