package categories

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joefazee/neo/app/api"
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

// GetCategoriesByCountry godoc
// @Summary      List categories by country
// @Description  Get a list of all categories for a specific country, with an option to fetch only active ones.
// @Tags         Categories
// @Accept       json
// @Produce      json
// @Param        countryId path      string  true  "Country ID"
// @Param        active    query     boolean false "Filter for active categories"
// @Success      200       {object}  api.Response{data=[]CategoryResponse}
// @Failure      400       {object}  api.Response{error=api.ErrorInfo}
// @Failure      500       {object}  api.Response{error=api.ErrorInfo}
// @Router       /api/v1/categories/c/{countryId} [get]
func (h *Handler) GetCategoriesByCountry(c *gin.Context) {
	countryIDParam := c.Param("countryId")
	countryID, err := uuid.Parse(countryIDParam)
	if err != nil {
		api.BadRequestResponse(c, "Invalid country ID format")
		return
	}

	activeOnly := c.Query("active") == "true"

	var categories []CategoryResponse
	if activeOnly {
		categories, err = h.service.GetActiveCategoriesByCountry(c.Request.Context(), countryID)
	} else {
		categories, err = h.service.GetCategoriesByCountry(c.Request.Context(), countryID)
	}

	if err != nil {
		api.InternalErrorResponse(c, "Failed to fetch categories")
		return
	}

	meta := gin.H{
		"count":       len(categories),
		"country_id":  countryID,
		"active_only": activeOnly,
	}
	api.SuccessResponseWithMeta(c, http.StatusOK, "Categories retrieved successfully", categories, meta)
}

// GetCategoryByID godoc
// @Summary      Get a single category by ID
// @Description  Retrieves the details of a specific category using its UUID.
// @Tags         Categories
// @Accept       json
// @Produce      json
// @Param        id  path      string  true  "Category ID"
// @Success      200 {object}  api.Response{data=CategoryWithCountryResponse}
// @Failure      400 {object}  api.Response{error=api.ErrorInfo}
// @Failure      404 {object}  api.Response{error=api.ErrorInfo}
// @Failure      500 {object}  api.Response{error=api.ErrorInfo}
// @Router       /api/v1/categories/{id} [get]
func (h *Handler) GetCategoryByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		api.BadRequestResponse(c, "Invalid category ID format")
		return
	}

	category, err := h.service.GetCategoryByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Category")
			return
		}
		api.InternalErrorResponse(c, "Failed to fetch category")
		return
	}

	api.SuccessResponse(c, http.StatusOK, "Category retrieved successfully", category)
}

// GetCategoryBySlug godoc
// @Summary      Get a category by slug
// @Description  Retrieves a category using its slug for a specific country.
// @Tags         Categories
// @Accept       json
// @Produce      json
// @Param        countryId path      string  true  "Country ID"
// @Param        slug      path      string  true  "Category Slug"
// @Success      200       {object}  api.Response{data=CategoryWithCountryResponse}
// @Failure      400       {object}  api.Response{error=api.ErrorInfo}
// @Failure      404       {object}  api.Response{error=api.ErrorInfo}
// @Failure      500       {object}  api.Response{error=api.ErrorInfo}
// @Router       /api/v1/countries/{countryId}/categories/{slug} [get]
func (h *Handler) GetCategoryBySlug(c *gin.Context) {
	countryIDParam := c.Param("countryId")
	countryID, err := uuid.Parse(countryIDParam)
	if err != nil {
		api.BadRequestResponse(c, "Invalid country ID format")
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		api.BadRequestResponse(c, "Category slug is required")
		return
	}

	category, err := h.service.GetCategoryBySlug(c.Request.Context(), countryID, slug)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Category")
			return
		}
		api.InternalErrorResponse(c, "Failed to fetch category")
		return
	}

	api.SuccessResponse(c, http.StatusOK, "Category retrieved successfully", category)
}

// CreateCategory godoc
// @Summary      Create a new category
// @Description  Adds a new category for a specific country.
// @Tags         Categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body      CreateCategoryRequest true  "Create Category Request"
// @Success      201       {object}  api.Response{data=CategoryResponse}
// @Failure      400       {object}  api.Response{error=api.ErrorInfo}
// @Failure      500       {object}  api.Response{error=api.ErrorInfo}
// @Router       /api/v1/categories [post]
func (h *Handler) CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	category, err := h.service.CreateCategory(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCountryID) ||
			errors.Is(err, models.ErrInvalidCategoryName) ||
			errors.Is(err, models.ErrInvalidCategorySlug) {
			api.BadRequestResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to create category")
		return
	}

	api.CreatedResponse(c, "Category created successfully", category)
}

// UpdateCategory godoc
// @Summary      Update a category
// @Description  Updates an existing category's details.
// @Tags         Categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      string               true  "Category ID"
// @Param        request body      UpdateCategoryRequest true  "Update Category Request"
// @Success      200     {object}  api.Response{data=CategoryResponse}
// @Failure      400     {object}  api.Response{error=api.ErrorInfo}
// @Failure      404     {object}  api.Response{error=api.ErrorInfo}
// @Failure      500     {object}  api.Response{error=api.ErrorInfo}
// @Router       /api/v1/categories/{id} [put]
func (h *Handler) UpdateCategory(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		api.BadRequestResponse(c, "Invalid category ID format")
		return
	}

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	category, err := h.service.UpdateCategory(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Category")
			return
		}
		if errors.Is(err, models.ErrInvalidCategoryName) {
			api.BadRequestResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to update category")
		return
	}

	api.UpdatedResponse(c, "Category updated successfully", category)
}

// DeleteCategory godoc
// @Summary      Delete a category
// @Description  Deletes a category by its ID.
// @Tags         Categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id  path      string  true  "Category ID"
// @Success      204 {object}  api.Response
// @Failure      400 {object}  api.Response{error=api.ErrorInfo}
// @Failure      404 {object}  api.Response{error=api.ErrorInfo}
// @Failure      500 {object}  api.Response{error=api.ErrorInfo}
// @Router       /api/v1/categories/{id} [delete]
func (h *Handler) DeleteCategory(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		api.BadRequestResponse(c, "Invalid category ID format")
		return
	}

	err = h.service.DeleteCategory(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Category")
			return
		}
		api.InternalErrorResponse(c, "Failed to delete category")
		return
	}

	c.Status(http.StatusNoContent)
}
