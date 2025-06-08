package prediction

import (
	"errors"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/joefazee/neo/app/api"
	"github.com/joefazee/neo/models"
)

// Handler handles HTTP requests for betting operations
type Handler struct {
	service   Service
	validator *validator.Validate
}

// NewHandler creates a new betting handler
func NewHandler(service Service) *Handler {
	return &Handler{
		service:   service,
		validator: validator.New(),
	}
}

// PlaceBet godoc
// @Summary Place a bet
// @Description Place a bet on a market outcome
// @Tags betting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body PlaceBetRequest true "Bet placement request"
// @Success 201 {object} api.Response{data=BetResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 403 {object} api.Response{error=api.ErrorInfo}
// @Failure 429 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/bets [post]
func (h *Handler) PlaceBet(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		api.UnauthorizedResponse(c)
		return
	}

	var req PlaceBetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ValidationErrorResponse(c, h.formatValidationErrors(err))
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		api.ValidationErrorResponse(c, h.formatValidationErrors(err))
		return
	}

	bet, err := h.service.PlaceBet(c.Request.Context(), userID, &req)
	if err != nil {
		if h.isBettingError(err) {
			api.ErrorResponse(c, 400, "BETTING_ERROR", err.Error(), nil)
			return
		}
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Market or outcome")
			return
		}
		if h.isRateLimitError(err) {
			api.ErrorResponse(c, 429, "RATE_LIMIT_EXCEEDED", err.Error(), nil)
			return
		}
		api.InternalErrorResponse(c, "Failed to place bet")
		return
	}

	api.CreatedResponse(c, "Bet placed successfully", bet)
}

// GetBetQuote godoc
// @Summary Get bet quote
// @Description Get a quote for a potential bet without placing it
// @Tags betting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body BetQuoteRequest true "Bet quote request"
// @Success 200 {object} api.Response{data=BetQuoteResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/bets/quote [post]
func (h *Handler) GetBetQuote(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		api.UnauthorizedResponse(c)
		return
	}

	var req BetQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ValidationErrorResponse(c, h.formatValidationErrors(err))
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		api.ValidationErrorResponse(c, h.formatValidationErrors(err))
		return
	}

	quote, err := h.service.CalculateBetQuote(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Market or outcome")
			return
		}
		api.InternalErrorResponse(c, "Failed to calculate bet quote")
		return
	}

	api.SuccessResponse(c, 200, "Bet quote calculated successfully", quote)
}

// GetMyBets godoc
// @Summary Get user bets
// @Description Get paginated list of user's bets with optional filters
// @Tags betting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param market_id query string false "Filter by market ID"
// @Param outcome_id query string false "Filter by outcome ID"
// @Param status query string false "Filter by bet status" Enums(active,settled,refunded)
// @Param date_from query string false "Filter bets from date (RFC3339)"
// @Param date_to query string false "Filter bets to date (RFC3339)"
// @Param min_amount query number false "Minimum bet amount"
// @Param max_amount query number false "Maximum bet amount"
// @Param sort_by query string false "Sort field" Enums(created_at,amount,price_per_contract) default(created_at)
// @Param sort_order query string false "Sort direction" Enums(asc,desc) default(desc)
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} api.Response{data=BetListResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/bets [get]
func (h *Handler) GetMyBets(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		api.UnauthorizedResponse(c)
		return
	}

	var filters BetFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		api.ValidationErrorResponse(c, err.Error())
		return
	}

	result, err := h.service.GetUserBets(c.Request.Context(), userID, &filters)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to fetch bets")
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

	api.SuccessResponseWithMeta(c, 200, "Bets retrieved successfully", result.Bets, meta)
}

// GetBetByID godoc
// @Summary Get bet details
// @Description Get detailed information about a specific bet
// @Tags betting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bet ID"
// @Success 200 {object} api.Response{data=BetResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 403 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/bets/{id} [get]
func (h *Handler) GetBetByID(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		api.UnauthorizedResponse(c)
		return
	}

	idParam := c.Param("id")
	betID, err := uuid.Parse(idParam)
	if err != nil {
		api.ValidationErrorResponse(c, "Invalid bet ID format")
		return
	}

	bet, err := h.service.GetBetByID(c.Request.Context(), userID, betID)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Bet")
			return
		}
		if errors.Is(err, models.ErrForbidden) {
			api.ForbiddenResponse(c, "Access denied to this bet")
			return
		}
		api.InternalErrorResponse(c, "Failed to fetch bet")
		return
	}

	api.SuccessResponse(c, 200, "Bet retrieved successfully", bet)
}

// CancelBet godoc
// @Summary Cancel a bet
// @Description Cancel an active bet (if within cancellation period)
// @Tags betting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bet ID"
// @Success 200 {object} api.Response
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 403 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/bets/{id}/cancel [post]
func (h *Handler) CancelBet(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		api.UnauthorizedResponse(c)
		return
	}

	idParam := c.Param("id")
	betID, err := uuid.Parse(idParam)
	if err != nil {
		api.ValidationErrorResponse(c, "Invalid bet ID format")
		return
	}

	err = h.service.CancelBet(c.Request.Context(), userID, betID)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Bet")
			return
		}
		if errors.Is(err, models.ErrForbidden) {
			api.ForbiddenResponse(c, "Access denied to this bet")
			return
		}
		if strings.Contains(err.Error(), "cannot be canceled") || strings.Contains(err.Error(), "cancellation period") {
			api.ErrorResponse(c, 400, "CANCELLATION_NOT_ALLOWED", err.Error(), nil)
			return
		}
		api.InternalErrorResponse(c, "Failed to cancel bet")
		return
	}

	api.SuccessResponse(c, 200, "Bet canceled successfully", nil)
}

// GetMyPositions godoc
// @Summary Get user positions
// @Description Get user's current positions across all markets
// @Tags betting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response{data=[]PositionResponse}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/bets/positions [get]
func (h *Handler) GetMyPositions(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		api.UnauthorizedResponse(c)
		return
	}

	positions, err := h.service.GetUserPositions(c.Request.Context(), userID)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to fetch positions")
		return
	}

	api.ListResponse(c, "Positions retrieved successfully", positions, len(positions))
}

// GetMyPortfolio godoc
// @Summary Get user portfolio
// @Description Get user's complete betting portfolio with statistics
// @Tags betting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response{data=PortfolioResponse}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/bets/portfolio [get]
func (h *Handler) GetMyPortfolio(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		api.UnauthorizedResponse(c)
		return
	}

	portfolio, err := h.service.GetUserPortfolio(c.Request.Context(), userID)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to fetch portfolio")
		return
	}

	api.SuccessResponse(c, 200, "Portfolio retrieved successfully", portfolio)
}

// GetMyStats godoc
// @Summary Get betting statistics
// @Description Get detailed betting statistics and performance metrics
// @Tags betting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response{data=BettingStatsResponse}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/bets/stats [get]
func (h *Handler) GetMyStats(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		api.UnauthorizedResponse(c)
		return
	}

	stats, err := h.service.GetUserBettingStats(c.Request.Context(), userID)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to fetch betting statistics")
		return
	}

	api.SuccessResponse(c, 200, "Betting statistics retrieved successfully", stats)
}

// GetPriceImpact godoc
// @Summary Get price impact analysis
// @Description Analyze how a potential bet would affect market prices
// @Tags betting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param market_id path string true "Market ID"
// @Param outcome_id path string true "Outcome ID"
// @Param amount query number true "Bet amount"
// @Success 200 {object} api.Response{data=PriceImpactResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/bets/markets/{market_id}/outcomes/{outcome_id}/price-impact [get]
func (h *Handler) GetPriceImpact(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		api.UnauthorizedResponse(c)
		return
	}

	marketIDParam := c.Param("market_id")
	marketID, err := uuid.Parse(marketIDParam)
	if err != nil {
		api.ValidationErrorResponse(c, "Invalid market ID format")
		return
	}

	outcomeIDParam := c.Param("outcome_id")
	outcomeID, err := uuid.Parse(outcomeIDParam)
	if err != nil {
		api.ValidationErrorResponse(c, "Invalid outcome ID format")
		return
	}

	amountStr := c.Query("amount")
	if amountStr == "" {
		api.ValidationErrorResponse(c, "Amount parameter is required")
		return
	}

	amount, err := parseDecimal(amountStr)
	if err != nil {
		api.ValidationErrorResponse(c, "Invalid amount format")
		return
	}

	impact, err := h.service.GetMarketPriceImpact(c.Request.Context(), marketID, outcomeID, amount)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Market or outcome")
			return
		}
		api.InternalErrorResponse(c, "Failed to calculate price impact")
		return
	}

	api.SuccessResponse(c, 200, "Price impact calculated successfully", impact)
}

// Helper methods

func (h *Handler) getUserIDFromContext(c *gin.Context) uuid.UUID {
	// TODO: Extract user ID from JWT context
	// For now, return a dummy UUID for testing
	// In production, this would extract from the JWT token
	if userIDStr, exists := c.Get("user_id"); exists {
		if userID, ok := userIDStr.(uuid.UUID); ok {
			return userID
		}
	}
	return uuid.Nil
}

func (h *Handler) formatValidationErrors(err error) interface{} {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		errors := make(map[string]string)
		for _, fieldError := range validationErrors {
			errors[fieldError.Field()] = h.getValidationMessage(fieldError)
		}
		return errors
	}
	return err.Error()
}

func (h *Handler) getValidationMessage(fieldError validator.FieldError) string {
	switch fieldError.Tag() {
	case "required":
		return "This field is required"
	case "gt":
		return "Value must be greater than " + fieldError.Param()
	case "gte":
		return "Value must be greater than or equal to " + fieldError.Param()
	case "lt":
		return "Value must be less than " + fieldError.Param()
	case "lte":
		return "Value must be less than or equal to " + fieldError.Param()
	case "min":
		return "Value must be at least " + fieldError.Param()
	case "max":
		return "Value must be at most " + fieldError.Param()
	default:
		return "Invalid value"
	}
}

func (h *Handler) isBettingError(err error) bool {
	return errors.Is(err, models.ErrSlippageExceeded) ||
		errors.Is(err, models.ErrPositionLimitExceeded) ||
		errors.Is(err, models.ErrMarketNotOpenForBetting) ||
		errors.Is(err, models.ErrInsufficientWalletBalance) ||
		errors.Is(err, models.ErrBetTooSmall) ||
		errors.Is(err, models.ErrBetTooLarge) ||
		errors.Is(err, models.ErrDailyLimitExceeded) ||
		strings.Contains(err.Error(), "betting") ||
		strings.Contains(err.Error(), "slippage") ||
		strings.Contains(err.Error(), "limit")
}

func (h *Handler) isRateLimitError(err error) bool {
	return errors.Is(err, models.ErrRateLimitExceeded) ||
		errors.Is(err, models.ErrBetCooldownActive)
}

func parseDecimal(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(s)
}
