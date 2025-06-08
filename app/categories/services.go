package categories

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/joefazee/neo/models"
)

// service implements the Service interface
type service struct {
	repo Repository
}

// NewService creates a new category service
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

// GetCategoriesByCountry returns all categories for a country
func (s *service) GetCategoriesByCountry(ctx context.Context, countryID uuid.UUID) ([]CategoryResponse, error) {
	categories, err := s.repo.GetByCountryID(ctx, countryID)
	if err != nil {
		return nil, err
	}
	return ToCategoryResponseList(categories), nil
}

// GetActiveCategoriesByCountry returns all active categories for a country
func (s *service) GetActiveCategoriesByCountry(ctx context.Context, countryID uuid.UUID) ([]CategoryResponse, error) {
	categories, err := s.repo.GetActiveByCountryID(ctx, countryID)
	if err != nil {
		return nil, err
	}
	return ToCategoryResponseList(categories), nil
}

// GetCategoryByID returns a category by ID
func (s *service) GetCategoryByID(ctx context.Context, id uuid.UUID) (*CategoryResponse, error) {
	category, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, err
	}
	return ToCategoryResponse(category), nil
}

// GetCategoryBySlug returns a category by slug within a country
func (s *service) GetCategoryBySlug(ctx context.Context, countryID uuid.UUID, slug string) (*CategoryResponse, error) {
	category, err := s.repo.GetBySlug(ctx, countryID, slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, err
	}
	return ToCategoryResponse(category), nil
}

// CreateCategory creates a new category
func (s *service) CreateCategory(ctx context.Context, req CreateCategoryRequest) (*CategoryResponse, error) {
	// Validate and normalize slug
	slug := s.normalizeSlug(req.Slug)
	if !s.isValidSlug(slug) {
		return nil, models.ErrInvalidCategorySlug
	}

	// Check if slug already exists in this country
	existing, err := s.repo.GetBySlug(ctx, req.CountryID, slug)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("category with this slug already exists in this country")
	}

	category := &models.Category{
		CountryID:   req.CountryID,
		Name:        strings.TrimSpace(req.Name),
		Slug:        slug,
		Description: strings.TrimSpace(req.Description),
		IsActive:    true,
		SortOrder:   req.SortOrder,
	}

	if err := category.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, category); err != nil {
		return nil, err
	}

	return ToCategoryResponse(category), nil
}

// UpdateCategory updates an existing category
func (s *service) UpdateCategory(ctx context.Context, id uuid.UUID, req UpdateCategoryRequest) (*CategoryResponse, error) {
	category, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		category.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		category.Description = strings.TrimSpace(*req.Description)
	}
	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}
	if req.SortOrder != nil {
		category.SortOrder = *req.SortOrder
	}

	if err := category.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, category); err != nil {
		return nil, err
	}

	return ToCategoryResponse(category), nil
}

// DeleteCategory deletes a category
func (s *service) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	// Check if category exists
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ErrRecordNotFound
		}
		return err
	}

	return s.repo.Delete(ctx, id)
}

// normalizeSlug normalizes a slug to lowercase and replaces spaces with hyphens
func (s *service) normalizeSlug(slug string) string {
	// Convert to lowercase
	slug = strings.ToLower(strings.TrimSpace(slug))

	// Replace spaces and underscores with hyphens
	slug = regexp.MustCompile(`[\s_]+`).ReplaceAllString(slug, "-")

	// Remove multiple consecutive hyphens
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	return slug
}

// isValidSlug checks if a slug contains only valid characters
func (s *service) isValidSlug(slug string) bool {
	if slug == "" {
		return false
	}

	// Allow only lowercase letters, numbers, and hyphens
	matched, _ := regexp.MatchString(`^[a-z0-9-]+$`, slug)
	return matched && !strings.HasPrefix(slug, "-") && !strings.HasSuffix(slug, "-")
}
