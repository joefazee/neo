package countries

import (
	"errors"

	"github.com/joefazee/neo/app/api"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/joefazee/neo/models"
)

// Handler handles HTTP requests for countries
type Handler struct {
	service Service
}

// NewHandler creates a new country handler
func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// GetAllCountries godoc
// @Summary List all countries
// @Description Get a list of all countries available on the platform
// @Tags countries
// @Accept json
// @Produce json
// @Success 200 {object} api.Response{data=[]CountryResponse}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/countries [get]
func (h *Handler) GetAllCountries(c *gin.Context) {
	countries, err := h.service.GetAllCountries(c.Request.Context())
	if err != nil {
		api.InternalErrorResponse(c, "Failed to fetch countries")
		return
	}

	api.ListResponse(c, "Countries retrieved successfully", countries, len(countries))
}

// GetActiveCountries godoc
// @Summary List active countries
// @Description Get a list of all active countries available on the platform
// @Tags countries
// @Accept json
// @Produce json
// @Success 200 {object} api.Response{data=[]CountryResponse}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/countries/active [get]
func (h *Handler) GetActiveCountries(c *gin.Context) {
	countries, err := h.service.GetActiveCountries(c.Request.Context())
	if err != nil {
		api.InternalErrorResponse(c, "Failed to fetch active countries")
		return
	}

	api.ListResponse(c, "Active countries retrieved successfully", countries, len(countries))
}

// GetCountryByID godoc
// @Summary Get country by ID
// @Description Get detailed information about a specific country
// @Tags countries
// @Accept json
// @Produce json
// @Param id path string true "Country ID"
// @Success 200 {object} api.Response{data=CountryResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/countries/{id} [get]
func (h *Handler) GetCountryByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		api.ValidationErrorResponse(c, "Invalid country ID format")
		return
	}

	country, err := h.service.GetCountryByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Country")
			return
		}
		api.InternalErrorResponse(c, "Failed to fetch country")
		return
	}

	api.SuccessResponse(c, 200, "Country retrieved successfully", country)
}

// GetCountryByCode godoc
// @Summary Get country by code
// @Description Get detailed information about a country using its 3-letter code
// @Tags countries
// @Accept json
// @Produce json
// @Param code path string true "Country Code (3 letters)"
// @Success 200 {object} api.Response{data=CountryResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/countries/code/{code} [get]
func (h *Handler) GetCountryByCode(c *gin.Context) {
	code := c.Param("code")
	if len(code) != 3 {
		api.ValidationErrorResponse(c, "Country code must be 3 characters")
		return
	}

	country, err := h.service.GetCountryByCode(c.Request.Context(), code)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Country")
			return
		}
		api.InternalErrorResponse(c, "Failed to fetch country")
		return
	}

	api.SuccessResponse(c, 200, "Country retrieved successfully", country)
}

// CreateCountry godoc
// @Summary Create a new country
// @Description Create a new country configuration
// @Tags countries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateCountryRequest true "Country creation request"
// @Success 201 {object} api.Response{data=CountryResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/countries [post]
func (h *Handler) CreateCountry(c *gin.Context) {
	var req CreateCountryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ValidationErrorResponse(c, err.Error())
		return
	}

	country, err := h.service.CreateCountry(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCountryCode) ||
			errors.Is(err, models.ErrInvalidCountryName) ||
			errors.Is(err, models.ErrInvalidCurrencyCode) ||
			errors.Is(err, models.ErrInvalidCurrencySymbol) {
			api.ValidationErrorResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to create country")
		return
	}

	api.CreatedResponse(c, "Country created successfully", country)
}

// UpdateCountry godoc
// @Summary Update a country
// @Description Update an existing country configuration
// @Tags countries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Country ID"
// @Param request body UpdateCountryRequest true "Country update request"
// @Success 200 {object} api.Response{data=CountryResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/countries/{id} [put]
func (h *Handler) UpdateCountry(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		api.ValidationErrorResponse(c, "Invalid country ID format")
		return
	}

	var req UpdateCountryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ValidationErrorResponse(c, err.Error())
		return
	}

	country, err := h.service.UpdateCountry(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Country")
			return
		}
		if errors.Is(err, models.ErrInvalidCountryName) ||
			errors.Is(err, models.ErrInvalidCurrencySymbol) {
			api.ValidationErrorResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to update country")
		return
	}

	api.UpdatedResponse(c, "Country updated successfully", country)
}

// DeleteCountry godoc
// @Summary Delete a country
// @Description Delete a country configuration
// @Tags countries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Country ID"
// @Success 204 {object} api.Response
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/countries/{id} [delete]
func (h *Handler) DeleteCountry(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		api.ValidationErrorResponse(c, "Invalid country ID format")
		return
	}

	err = h.service.DeleteCountry(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Country")
			return
		}
		api.InternalErrorResponse(c, "Failed to delete country")
		return
	}

	api.DeletedResponse(c, "Country deleted successfully")
}
