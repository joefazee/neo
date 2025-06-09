package markets

import (
	"errors"
	"net/http"
	"strings"

	"github.com/joefazee/neo/app/countries"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/joefazee/neo/app/api"
	"github.com/joefazee/neo/app/categories"
	"github.com/joefazee/neo/internal/sanitizer"
	"github.com/joefazee/neo/internal/validator"
	"github.com/joefazee/neo/models"
)

// Handler handles HTTP requests for markets
type Handler struct {
	service      Service
	countryRepo  countries.Repository
	categoryRepo categories.Repository
	sanitizer    sanitizer.HTMLStripperer
}

// NewHandler creates a new market handler
func NewHandler(service Service, countryRepo countries.Repository, categoryRepo categories.Repository, sanitizer sanitizer.HTMLStripperer) *Handler {
	return &Handler{
		service:      service,
		countryRepo:  countryRepo,
		categoryRepo: categoryRepo,
		sanitizer:    sanitizer,
	}
}

// parseUUIDFromParam extracts and validates UUID from path parameter
func (h *Handler) parseUUIDFromParam(c *gin.Context, paramName string) (uuid.UUID, bool) {
	param := c.Param(paramName)
	id, err := uuid.Parse(param)
	if err != nil {
		api.BadRequestResponse(c, "Invalid "+paramName+" format")
		return uuid.Nil, false
	}
	return id, true
}

// bindJSONRequest binds JSON request body to the provided struct
func (h *Handler) bindJSONRequest(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return false
	}
	return true
}

// handleServiceError handles common service errors with appropriate responses
func (h *Handler) handleServiceError(c *gin.Context, err error, entityName, operation string) {
	if errors.Is(err, models.ErrRecordNotFound) {
		api.NotFoundResponse(c, entityName)
		return
	}
	if h.isValidationError(err) {
		api.BadRequestResponse(c, err.Error())
		return
	}
	api.InternalErrorResponse(c, "Failed to "+operation)
}

// executeWithUUIDAndServiceCall is a generic helper for handlers that need UUID parsing and a service call
func (h *Handler) executeWithUUIDAndServiceCall(
	c *gin.Context,
	paramName string,
	entityName string,
	operation string,
	serviceCall func(uuid.UUID) (interface{}, error),
	successMessage string,
) {
	id, ok := h.parseUUIDFromParam(c, paramName)
	if !ok {
		return
	}

	result, err := serviceCall(id)
	if err != nil {
		h.handleServiceError(c, err, entityName, operation)
		return
	}

	api.SuccessResponse(c, 200, successMessage, result)
}

// GetMarkets godoc
// @Summary List prediction markets
// @Description Get a paginated list of prediction markets with optional filters
// @Tags markets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param country_id query string false "Filter by country ID"
// @Param category_id query string false "Filter by category ID"
// @Param creator_id query string false "Filter by creator ID"
// @Param status query string false "Filter by market status" Enums(draft,open,closed,resolved,voided)
// @Param market_type query string false "Filter by market type" Enums(binary,multi_outcome)
// @Param search query string false "Search in title and description"
// @Param tags query string false "Filter by tags (comma-separated)"
// @Param sort_by query string false "Sort field" Enums(created_at,updated_at,close_time,total_pool_amount,title) default(created_at)
// @Param sort_order query string false "Sort direction" Enums(asc,desc) default(desc)
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} api.Response{data=MarketListResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/markets [get]
func (h *Handler) GetMarkets(c *gin.Context) {
	var filters MarketFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	result, err := h.service.GetMarkets(c.Request.Context(), &filters)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to fetch markets")
		return
	}

	if len(result.Markets) == 0 {
		api.SuccessResponseWithMeta(c, http.StatusOK, "No markets found", nil, api.PaginationMeta{})
		return
	}

	meta := api.PaginationMeta{
		Page:       result.Page,
		PerPage:    result.PerPage,
		Total:      result.Total,
		TotalPages: int((result.Total + int64(result.PerPage) - 1) / int64(result.PerPage)),
		HasNext:    int64(result.Page*result.PerPage) < result.Total,
		HasPrev:    result.Page > 1,
	}

	api.SuccessResponseWithMeta(c, 200, "Markets retrieved successfully", result.Markets, meta)
}

// GetMarketByID godoc
// @Summary Get market details
// @Description Get detailed information about a specific prediction market
// @Tags markets
// @Accept json
// @Produce json
// @Param id path string true "Market ID"
// @Success 200 {object} api.Response{data=MarketDetailResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/markets/{id} [get]
func (h *Handler) GetMarketByID(c *gin.Context) {
	h.executeWithUUIDAndServiceCall(
		c,
		"id",
		"Market",
		"fetch market",
		func(id uuid.UUID) (interface{}, error) {
			return h.service.GetMarketByID(c.Request.Context(), id)
		},
		"Market retrieved successfully",
	)
}

// GetMarketsByCategory godoc
// @Summary Get markets by category
// @Description Get all markets for a specific category
// @Tags markets
// @Accept json
// @Produce json
// @Param category_id path string true "Category ID"
// @Success 200 {object} api.Response{data=[]MarketResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/markets/category/{category_id} [get]
func (h *Handler) GetMarketsByCategory(c *gin.Context) {
	categoryID, ok := h.parseUUIDFromParam(c, "category_id")
	if !ok {
		return
	}

	markets, err := h.service.GetMarketsByCategory(c.Request.Context(), categoryID)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to fetch markets")
		return
	}

	api.ListResponse(c, "Markets retrieved successfully", markets, len(markets))
}

// GetMyMarkets godoc
// @Summary Get my markets
// @Description Get markets created by the authenticated user
// @Tags markets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response{data=[]MarketResponse}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/markets/my [get]
func (h *Handler) GetMyMarkets(c *gin.Context) {
	// TODO: Extract user ID from JWT context
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		api.UnauthorizedResponse(c)
		return
	}

	markets, err := h.service.GetMyMarkets(c.Request.Context(), userID)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to fetch markets")
		return
	}

	api.ListResponse(c, "Your markets retrieved successfully", markets, len(markets))
}

// CreateMarket godoc
// @Summary Create a new market
// @Description Create a new prediction market
// @Tags markets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateMarketRequest true "Market creation request"
// @Success 201 {object} api.Response{data=MarketDetailResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/markets [post]
//
//	@Example request body {
//		"country_id": "550e8400-e29b-41d4-a716-446655440000",
//		"category_id": "550e8400-e29b-41d4-a716-446655440001",
//		"title": "Will OpenAI release GPT-5 by end of 2024?",
//		"description": "This market resolves to YES if OpenAI officially announces and releases GPT-5
//				(or equivalent next-generation model) by December 31, 2024. The model must be generally
//					available to the public, not just in limited beta. Announcements without actual release do not count.",
//		"market_type": "binary",
//		"close_time": "2024-12-31T23:59:59Z",
//		"resolution_deadline": "2025-01-07T23:59:59Z",
//		"min_bet_amount": "100.00",
//		"max_bet_amount": "10000.00",
//		"outcomes": [
//			{
//				"outcome_key": "yes",
//				"outcome_label": "Yes, GPT-5 will be released",
//				"sort_order": 1
//			},
//			{
//				"outcome_key": "no",
//				"outcome_label": "No, GPT-5 will not be released",
//				"sort_order": 2
//			}
//		],
//		"safeguard_config": {
//			"min_quorum_amount": "5000.00",
//			"min_outcomes": 2,
//			"house_bot_enabled": true,
//			"house_bot_amount": "2000.00",
//			"imbalance_threshold": "0.80",
//			"void_on_quorum_fail": true
//		},
//		"tags": ["ai", "openai", "gpt", "technology"]
//	}
func (h *Handler) CreateMarket(c *gin.Context) {
	var req CreateMarketRequest
	if !h.bindJSONRequest(c, &req) {
		return
	}

	// Validate the request
	v := validator.New()
	if !req.Validate(c.Request.Context(), v, h.countryRepo, h.categoryRepo, h.sanitizer) {
		api.ValidationErrorResponse(c, validator.NewValidationError("Validation failed", v.Errors))
		return
	}

	market, err := h.service.CreateMarket(c.Request.Context(), &req)
	if err != nil {
		h.handleServiceError(c, err, "Market", "create market")
		return
	}

	api.CreatedResponse(c, "Market created successfully", market)
}

// UpdateMarket godoc
// @Summary Update a market
// @Description Update an existing prediction market
// @Tags markets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Market ID"
// @Param request body UpdateMarketRequest true "Market update request"
// @Success 200 {object} api.Response{data=MarketDetailResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/markets/{id} [put]
func (h *Handler) UpdateMarket(c *gin.Context) {
	id, ok := h.parseUUIDFromParam(c, "id")
	if !ok {
		return
	}

	var req UpdateMarketRequest
	if !h.bindJSONRequest(c, &req) {
		return
	}

	market, err := h.service.UpdateMarket(c.Request.Context(), id, &req)
	if err != nil {
		h.handleServiceError(c, err, "Market", "update market")
		return
	}

	api.UpdatedResponse(c, "Market updated successfully", market)
}

// ResolveMarket godoc
// @Summary Resolve a market
// @Description Resolve a prediction market by declaring the winning outcome
// @Tags markets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Market ID"
// @Param request body ResolveMarketRequest true "Market resolution request"
// @Success 200 {object} api.Response{data=MarketDetailResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/markets/{id}/resolve [post]
func (h *Handler) ResolveMarket(c *gin.Context) {
	id, ok := h.parseUUIDFromParam(c, "id")
	if !ok {
		return
	}

	var req ResolveMarketRequest
	if !h.bindJSONRequest(c, &req) {
		return
	}

	market, err := h.service.ResolveMarket(c.Request.Context(), id, req)
	if err != nil {
		h.handleServiceError(c, err, "Market", "resolve market")
		return
	}

	api.SuccessResponse(c, 200, "Market resolved successfully", market)
}

// VoidMarket godoc
// @Summary Void a market
// @Description Void a prediction market and refund all bets
// @Tags markets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Market ID"
// @Param reason query string true "Reason for voiding the market"
// @Success 200 {object} api.Response
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/markets/{id}/void [post]
func (h *Handler) VoidMarket(c *gin.Context) {
	id, ok := h.parseUUIDFromParam(c, "id")
	if !ok {
		return
	}

	reason := c.Query("reason")
	if reason == "" {
		api.BadRequestResponse(c, "Void reason is required")
		return
	}

	err := h.service.VoidMarket(c.Request.Context(), id, reason)
	if err != nil {
		h.handleServiceError(c, err, "Market", "void market")
		return
	}

	api.SuccessResponse(c, 200, "Market voided successfully", nil)
}

// DeleteMarket godoc
// @Summary Delete a market
// @Description Delete a prediction market (only draft markets with no bets)
// @Tags markets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Market ID"
// @Success 204 {object} api.Response
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/markets/{id} [delete]
func (h *Handler) DeleteMarket(c *gin.Context) {
	id, ok := h.parseUUIDFromParam(c, "id")
	if !ok {
		return
	}

	err := h.service.DeleteMarket(c.Request.Context(), id)
	if err != nil {
		h.handleServiceError(c, err, "Market", "delete market")
		return
	}

	api.DeletedResponse(c, "Market deleted successfully")
}

// AddMarketOutcome godoc
// @Summary Add market outcome
// @Description Add a new outcome to an existing market
// @Tags markets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Market ID"
// @Param request body CreateOutcomeRequest true "Outcome creation request"
// @Success 201 {object} api.Response{data=OutcomeResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/markets/{id}/outcomes [post]
func (h *Handler) AddMarketOutcome(c *gin.Context) {
	marketID, ok := h.parseUUIDFromParam(c, "id")
	if !ok {
		return
	}

	var req CreateOutcomeRequest
	if !h.bindJSONRequest(c, &req) {
		return
	}

	outcome, err := h.service.AddMarketOutcome(c.Request.Context(), marketID, req)
	if err != nil {
		h.handleServiceError(c, err, "Market", "add outcome")
		return
	}

	api.CreatedResponse(c, "Outcome added successfully", outcome)
}

// UpdateMarketOutcome godoc
// @Summary Update market outcome
// @Description Update an existing market outcome
// @Tags markets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param outcome_id path string true "Outcome ID"
// @Param request body UpdateOutcomeRequest true "Outcome update request"
// @Success 200 {object} api.Response{data=OutcomeResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/outcomes/{outcome_id} [put]
func (h *Handler) UpdateMarketOutcome(c *gin.Context) {
	outcomeID, ok := h.parseUUIDFromParam(c, "outcome_id")
	if !ok {
		return
	}

	var req UpdateOutcomeRequest
	if !h.bindJSONRequest(c, &req) {
		return
	}

	outcome, err := h.service.UpdateMarketOutcome(c.Request.Context(), outcomeID, req)
	if err != nil {
		h.handleServiceError(c, err, "Outcome", "update outcome")
		return
	}

	api.UpdatedResponse(c, "Outcome updated successfully", outcome)
}

// DeleteMarketOutcome godoc
// @Summary Delete market outcome
// @Description Delete a market outcome
// @Tags markets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param outcome_id path string true "Outcome ID"
// @Success 204 {object} api.Response
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/outcomes/{outcome_id} [delete]
func (h *Handler) DeleteMarketOutcome(c *gin.Context) {
	outcomeID, ok := h.parseUUIDFromParam(c, "outcome_id")
	if !ok {
		return
	}

	err := h.service.DeleteMarketOutcome(c.Request.Context(), outcomeID)
	if err != nil {
		h.handleServiceError(c, err, "Outcome", "delete outcome")
		return
	}

	api.DeletedResponse(c, "Outcome deleted successfully")
}

// GetMarketPrices godoc
// @Summary Get market prices
// @Description Get current prices for all outcomes in a market
// @Tags markets
// @Accept json
// @Produce json
// @Param id path string true "Market ID"
// @Success 200 {object} api.Response{data=map[string]PriceInfo}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/markets/{id}/prices [get]
func (h *Handler) GetMarketPrices(c *gin.Context) {
	h.executeWithUUIDAndServiceCall(
		c,
		"id",
		"Market",
		"calculate prices",
		func(id uuid.UUID) (interface{}, error) {
			return h.service.CalculateCurrentPrices(c.Request.Context(), id)
		},
		"Market prices retrieved successfully",
	)
}

// GetMarketSafeguards godoc
// @Summary Get market safeguards status
// @Description Get current safeguard status and risk assessment for a market
// @Tags markets
// @Accept json
// @Produce json
// @Param id path string true "Market ID"
// @Success 200 {object} api.Response{data=SafeguardStatus}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/markets/{id}/safeguards [get]
func (h *Handler) GetMarketSafeguards(c *gin.Context) {
	h.executeWithUUIDAndServiceCall(
		c,
		"id",
		"Market",
		"check safeguards",
		func(id uuid.UUID) (interface{}, error) {
			return h.service.CheckSafeguards(c.Request.Context(), id)
		},
		"Safeguard status retrieved successfully",
	)
}

// Helper methods

func (h *Handler) getUserIDFromContext(_ *gin.Context) uuid.UUID {
	// TODO: Extract user ID from JWT context
	// For now, return nil UUID
	return uuid.Nil
}

func (h *Handler) isValidationError(err error) bool {
	// Check if error is a validation error
	return errors.Is(err, models.ErrInvalidMarketTitle) ||
		errors.Is(err, models.ErrInvalidMarketType) ||
		errors.Is(err, models.ErrInvalidCloseTime) ||
		errors.Is(err, models.ErrInvalidResolutionTime) ||
		errors.Is(err, models.ErrInvalidBetAmount) ||
		strings.Contains(err.Error(), "validation") ||
		strings.Contains(err.Error(), "invalid") ||
		strings.Contains(err.Error(), "required") ||
		strings.Contains(err.Error(), "must be")
}
