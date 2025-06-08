package categories

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/joefazee/neo/models"
)

// Handler handles HTTP requests for categories
type Handler struct {
	service Service
}

// NewHandler creates a new category handler
func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// GetCategoriesByCountry handles GET /countries/:countryId/categories
func (h *Handler) GetCategoriesByCountry(c *gin.Context) {
	countryIDParam := c.Param("countryId")
	countryID, err := uuid.Parse(countryIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid country ID format",
		})
		return
	}

	// Check if we should only return active categories
	activeOnly := c.Query("active") == "true"

	var categories []CategoryResponse
	if activeOnly {
		categories, err = h.service.GetActiveCategoriesByCountry(c.Request.Context(), countryID)
	} else {
		categories, err = h.service.GetCategoriesByCountry(c.Request.Context(), countryID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch categories",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": categories,
		"meta": gin.H{
			"count":       len(categories),
			"country_id":  countryID,
			"active_only": activeOnly,
		},
	})
}

// GetCategoryByID handles GET /categories/:id
func (h *Handler) GetCategoryByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid category ID format",
		})
		return
	}

	category, err := h.service.GetCategoryByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Category not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch category",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": category,
	})
}

// GetCategoryBySlug handles GET /countries/:countryId/categories/:slug
func (h *Handler) GetCategoryBySlug(c *gin.Context) {
	countryIDParam := c.Param("countryId")
	countryID, err := uuid.Parse(countryIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid country ID format",
		})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Category slug is required",
		})
		return
	}

	category, err := h.service.GetCategoryBySlug(c.Request.Context(), countryID, slug)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Category not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch category",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": category,
	})
}

// CreateCategory handles POST /categories
func (h *Handler) CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	category, err := h.service.CreateCategory(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCountryID) ||
			errors.Is(err, models.ErrInvalidCategoryName) ||
			errors.Is(err, models.ErrInvalidCategorySlug) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create category",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": category,
	})
}

// UpdateCategory handles PUT /categories/:id
func (h *Handler) UpdateCategory(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid category ID format",
		})
		return
	}

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	category, err := h.service.UpdateCategory(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Category not found",
			})
			return
		}
		if errors.Is(err, models.ErrInvalidCategoryName) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update category",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": category,
	})
}

// DeleteCategory handles DELETE /categories/:id
func (h *Handler) DeleteCategory(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid category ID format",
		})
		return
	}

	err = h.service.DeleteCategory(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Category not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete category",
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
