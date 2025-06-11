package wallet

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joefazee/neo/app/api"
	"github.com/joefazee/neo/models"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// CreateWallet godoc
// @Summary Create a new wallet
// @Description Create a new wallet for a user with specified currency
// @Tags wallets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateWalletRequest true "Wallet creation request"
// @Success 201 {object} api.Response{data=Response}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 409 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/wallets [post]
func (h *Handler) CreateWallet(c *gin.Context) {
	var req CreateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	wallet, err := h.service.CreateWallet(c.Request.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			api.ConflictResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to create wallet")
		return
	}

	api.CreatedResponse(c, "Wallet created successfully", wallet)
}

// GetWallet godoc
// @Summary Get wallet by ID
// @Description Get detailed information about a specific wallet
// @Tags wallets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Wallet ID"
// @Success 200 {object} api.Response{data=Response}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/wallets/{id} [get]
func (h *Handler) GetWallet(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.BadRequestResponse(c, "Invalid wallet ID format")
		return
	}

	wallet, err := h.service.GetWallet(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Wallet")
			return
		}
		api.InternalErrorResponse(c, "Failed to get wallet")
		return
	}

	api.SuccessResponse(c, 200, "Wallet retrieved successfully", wallet)
}

// GetUserWallets godoc
// @Summary Get user wallets
// @Description Get all wallets for a specific user
// @Tags wallets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Success 200 {object} api.Response{data=[]Response}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/users/{user_id}/wallets [get]
func (h *Handler) GetUserWallets(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		api.BadRequestResponse(c, "Invalid user ID format")
		return
	}

	wallets, err := h.service.GetUserWallets(c.Request.Context(), userID)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to get user wallets")
		return
	}

	api.SuccessResponse(c, 200, "User wallets retrieved successfully", wallets)
}

// CreditWallet godoc
// @Summary Credit wallet
// @Description Add funds to a wallet
// @Tags wallets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Wallet ID"
// @Param request body CreditWalletRequest true "Credit request"
// @Success 200 {object} api.Response{data=OperationResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/wallets/{id}/credit [post]
// //nolint: dupl
func (h *Handler) CreditWallet(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.BadRequestResponse(c, "Invalid wallet ID format")
		return
	}

	var req CreditWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	result, err := h.service.CreditWallet(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Wallet")
			return
		}
		if strings.Contains(err.Error(), "locked") {
			api.BadRequestResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to credit wallet")
		return
	}

	api.SuccessResponse(c, 200, "Wallet credited successfully", result)
}

// DebitWallet godoc
// @Summary Debit wallet
// @Description Remove funds from a wallet
// @Tags wallets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Wallet ID"
// @Param request body DebitWalletRequest true "Debit request"
// @Success 200 {object} api.Response{data=OperationResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/wallets/{id}/debit [post]
// //nolint: dupl
func (h *Handler) DebitWallet(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.BadRequestResponse(c, "Invalid wallet ID format")
		return
	}

	var req DebitWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	result, err := h.service.DebitWallet(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Wallet")
			return
		}
		if errors.Is(err, models.ErrInsufficientBalance) {
			api.BadRequestResponse(c, "Insufficient balance")
			return
		}
		if strings.Contains(err.Error(), "locked") {
			api.BadRequestResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to debit wallet")
		return
	}

	api.SuccessResponse(c, 200, "Wallet debited successfully", result)
}

// LockFunds godoc
// @Summary Lock funds in wallet
// @Description Lock specified amount in a wallet
// @Tags wallets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Wallet ID"
// @Param request body LockFundsRequest true "Lock funds request"
// @Success 200 {object} api.Response{data=OperationResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/wallets/{id}/lock-funds [post]
// //nolint: dupl
func (h *Handler) LockFunds(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.BadRequestResponse(c, "Invalid wallet ID format")
		return
	}

	var req LockFundsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	result, err := h.service.LockFunds(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Wallet")
			return
		}
		if errors.Is(err, models.ErrInsufficientBalance) {
			api.BadRequestResponse(c, "Insufficient available balance")
			return
		}
		if strings.Contains(err.Error(), "locked") {
			api.BadRequestResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to lock funds")
		return
	}

	api.SuccessResponse(c, 200, "Funds locked successfully", result)
}

// UnlockFunds godoc
// @Summary Unlock funds in wallet
// @Description Unlock specified amount in a wallet
// @Tags wallets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Wallet ID"
// @Param request body UnlockFundsRequest true "Unlock funds request"
// @Success 200 {object} api.Response{data=OperationResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/wallets/{id}/unlock-funds [post]
// //nolint: dupl
func (h *Handler) UnlockFunds(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.BadRequestResponse(c, "Invalid wallet ID format")
		return
	}

	var req UnlockFundsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	result, err := h.service.UnlockFunds(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Wallet")
			return
		}
		if strings.Contains(err.Error(), "locked") {
			api.BadRequestResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to unlock funds")
		return
	}

	api.SuccessResponse(c, 200, "Funds unlocked successfully", result)
}

// LockWallet godoc
// @Summary Lock/unlock wallet
// @Description Lock or unlock a wallet to prevent/allow operations
// @Tags wallets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Wallet ID"
// @Param request body LockWalletRequest true "Lock wallet request"
// @Success 200 {object} api.Response{data=Response}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/wallets/{id}/lock [patch]
func (h *Handler) LockWallet(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.BadRequestResponse(c, "Invalid wallet ID format")
		return
	}

	var req LockWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	wallet, err := h.service.LockWallet(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Wallet")
			return
		}
		api.InternalErrorResponse(c, "Failed to update wallet lock status")
		return
	}

	message := "Wallet unlocked successfully"
	if req.IsLocked {
		message = "Wallet locked successfully"
	}

	api.SuccessResponse(c, 200, message, wallet)
}

// GetWalletTransactions godoc
// @Summary Get wallet transactions
// @Description Get transaction history for a specific wallet
// @Tags wallets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Wallet ID"
// @Param limit query int false "Limit (default: 20, max: 100)"
// @Param offset query int false "Offset (default: 0)"
// @Success 200 {object} api.Response{data=[]TransactionResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/wallets/{id}/transactions [get]
func (h *Handler) GetWalletTransactions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.BadRequestResponse(c, "Invalid wallet ID format")
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil {
			limit = parsedLimit
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil {
			offset = parsedOffset
		}
	}

	transactions, err := h.service.GetWalletTransactions(c.Request.Context(), id, limit, offset)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to get wallet transactions")
		return
	}

	api.SuccessResponse(c, 200, "Wallet transactions retrieved successfully", transactions)
}
