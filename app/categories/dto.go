package categories

import (
	"time"

	"github.com/google/uuid"
	"github.com/joefazee/neo/models"
)

// CreateCategoryRequest represents the request to create a category
type CreateCategoryRequest struct {
	CountryID   uuid.UUID `json:"country_id" binding:"required"`
	Name        string    `json:"name" binding:"required,min=2,max=100"`
	Slug        string    `json:"slug" binding:"required,min=2,max=100,lowercase"`
	Description string    `json:"description,omitempty" binding:"omitempty,max=500"`
	SortOrder   int       `json:"sort_order,omitempty"`
}

// UpdateCategoryRequest represents the request to update a category
type UpdateCategoryRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=2,max=100"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=500"`
	IsActive    *bool   `json:"is_active,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

// CategoryResponse represents the response for category data
type CategoryResponse struct {
	ID          uuid.UUID `json:"id"`
	CountryID   uuid.UUID `json:"country_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CategoryWithCountryResponse represents category data with country information
type CategoryWithCountryResponse struct {
	ID          uuid.UUID         `json:"id"`
	CountryID   uuid.UUID         `json:"country_id"`
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	Description string            `json:"description"`
	IsActive    bool              `json:"is_active"`
	SortOrder   int               `json:"sort_order"`
	Country     *CountryReference `json:"country,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// CountryReference represents a minimal country reference
type CountryReference struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Code           string    `json:"code"`
	CurrencyCode   string    `json:"currency_code"`
	CurrencySymbol string    `json:"currency_symbol"`
}

// ToCategoryResponse converts a models.Category to CategoryResponse
func ToCategoryResponse(category *models.Category) *CategoryResponse {
	return &CategoryResponse{
		ID:          category.ID,
		CountryID:   category.CountryID,
		Name:        category.Name,
		Slug:        category.Slug,
		Description: category.Description,
		IsActive:    category.IsActive,
		SortOrder:   category.SortOrder,
		CreatedAt:   category.CreatedAt,
		UpdatedAt:   category.UpdatedAt,
	}
}

// ToCategoryWithCountryResponse converts a models.Category with Country to CategoryWithCountryResponse
func ToCategoryWithCountryResponse(category *models.Category) *CategoryWithCountryResponse {
	response := &CategoryWithCountryResponse{
		ID:          category.ID,
		CountryID:   category.CountryID,
		Name:        category.Name,
		Slug:        category.Slug,
		Description: category.Description,
		IsActive:    category.IsActive,
		SortOrder:   category.SortOrder,
		CreatedAt:   category.CreatedAt,
		UpdatedAt:   category.UpdatedAt,
	}

	if category.Country != nil {
		response.Country = &CountryReference{
			ID:             category.Country.ID,
			Name:           category.Country.Name,
			Code:           category.Country.Code,
			CurrencyCode:   category.Country.CurrencyCode,
			CurrencySymbol: category.Country.CurrencySymbol,
		}
	}

	return response
}

// ToCategoryResponseList converts a slice of models.Category to CategoryResponse
func ToCategoryResponseList(categories []models.Category) []CategoryResponse {
	responses := make([]CategoryResponse, len(categories))
	for i := range categories {
		responses[i] = *ToCategoryResponse(&categories[i])
	}
	return responses
}

// ToCategoryWithCountryResponseList converts a slice of models.Category to CategoryWithCountryResponse
func ToCategoryWithCountryResponseList(categories []models.Category) []CategoryWithCountryResponse {
	responses := make([]CategoryWithCountryResponse, len(categories))
	for i := range categories {
		responses[i] = *ToCategoryWithCountryResponse(&categories[i])
	}
	return responses
}
