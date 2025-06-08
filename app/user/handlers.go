package user

import (
	"net/http"

	"github.com/joefazee/neo/app/countries"
	"github.com/joefazee/neo/internal/validator"

	"github.com/gin-gonic/gin"
	"github.com/joefazee/neo/app/api"
)

// Handler handles HTTP requests for user operations
type Handler struct {
	service           Service
	countryRepository countries.Repository
}

// NewHandler creates a new user handler
func NewHandler(service Service, countryRepository countries.Repository) *Handler {
	return &Handler{service: service, countryRepository: countryRepository}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request  body      RegisterUserRequest  true  "User registration details"
// @Success      201      {object}  api.Response{data=Response}
// @Failure      400      {object}  api.Response{error=api.ErrorInfo}
// @Failure      500      {object}  api.Response{error=api.ErrorInfo}
// @Router       /api/v1/users/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	v := validator.New()
	if !req.Validate(c, v, h.countryRepository) {
		api.ValidationErrorResponse(c, validator.NewValidationError("Validation failed", v.Errors))
		return
	}

	user, err := h.service.Register(c.Request.Context(), &req)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to register user")
		return
	}

	api.CreatedResponse(c, "User registered successfully", user)
}

// Login godoc
// @Summary      Log in a user
// @Description  Authenticate a user and return an access token
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "User credentials"
// @Success      200      {object}  api.Response{data=LoginResponse}
// @Failure      400      {object}  api.Response{error=api.ErrorInfo}
// @Failure      401      {object}  api.Response{error=api.ErrorInfo}
// @Failure      500      {object}  api.Response{error=api.ErrorInfo}
// @Router       /api/v1/users/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	resp, err := h.service.Login(c.Request.Context(), &req)
	if err != nil {
		api.UnauthorizedResponse(c)
		return
	}

	api.SuccessResponse(c, http.StatusOK, "Login successful", resp)
}

// RequestPasswordReset godoc
// @Summary      Request a password reset
// @Description  Send a password reset email if the user exists
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request  body      PasswordResetRequest  true  "Email for password reset"
// @Success      200      {object}  api.Response
// @Failure      400      {object}  api.Response{error=api.ErrorInfo}
// @Failure      500      {object}  api.Response{error=api.ErrorInfo}
// @Router       /api/v1/users/password-reset [post]
func (h *Handler) RequestPasswordReset(c *gin.Context) {
	var req PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		api.InternalErrorResponse(c, "Failed to process request")
		return
	}

	api.SuccessResponse(c, http.StatusOK, "Password reset email sent", nil)
}

// ResetPassword godoc
// @Summary      Reset a user's password
// @Description  Set a new password using a valid reset token
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request  body      SetNewPasswordRequest  true  "Token and new password"
// @Success      200      {object}  api.Response
// @Failure      400      {object}  api.Response{error=api.ErrorInfo}
// @Failure      500      {object}  api.Response{error=api.ErrorInfo}
// @Router       /api/v1/users/reset-password [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	var req SetNewPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		api.InternalErrorResponse(c, "Failed to reset password")
		return
	}

	api.SuccessResponse(c, http.StatusOK, "Password reset successfully", nil)
}
