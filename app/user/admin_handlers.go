package user

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joefazee/neo/app/api"
	"github.com/joefazee/neo/internal/logger"
	"github.com/joefazee/neo/internal/sanitizer"
	"github.com/joefazee/neo/internal/validator"
)

type AdminHandler struct {
	service   AdminService
	sanitizer sanitizer.HTMLStripperer
	logger    logger.Logger
}

func NewAdminHandler(service AdminService, sanitizer sanitizer.HTMLStripperer, logger logger.Logger) *AdminHandler {
	return &AdminHandler{service: service, sanitizer: sanitizer, logger: logger}
}

// GetUsers godoc
// @Summary      List users (Admin)
// @Description  Retrieves a paginated list of users with filtering options.
// @Tags         Admin
// @Produce      json
// @Param        page    query     int     false  "Page number" default(1)
// @Param        per_page query    int     false  "Items per page" default(20)
// @Param        status  query     string  false  "Filter by status (active or inactive)" Enums(active, inactive)
// @Param        search  query     string  false  "Search term for name or email"
// @Param        sort_by query     string  false  "Sort by field" Enums(created_at, first_name, email)
// @Param        sort_order query string  false  "Sort order" Enums(asc, desc)
// @Security     BearerAuth
// @Success      200  {object}  api.Response{data=[]AdminUserResponse}
// @Failure      400  {object}  api.Response{error=api.ErrorInfo}
// @Failure      500  {object}  api.Response{error=api.ErrorInfo}
// @Router       /api/v1/admin/users [get]
func (h *AdminHandler) GetUsers(c *gin.Context) {
	var filters AdminUserFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	v := validator.New()
	filters.SanitizeAndValidate(v, h.sanitizer)
	if !v.Valid() {
		api.ValidationErrorResponse(c, validator.NewValidationError("Validation failed", v.Errors))
		return
	}

	users, total, err := h.service.GetUsers(c.Request.Context(), &filters)
	if err != nil {
		h.logger.Error(err, logger.Fields{"handler": "GetUsers"})
		api.InternalErrorResponse(c, "Failed to retrieve users")
		return
	}
	api.PaginatedResponse(c, "Users retrieved successfully", users, api.PaginationMeta{
		Page:    filters.Page,
		PerPage: filters.PerPage,
		Total:   total,
	})
}

// UpdateUserStatus godoc
// @Summary      Update user status (Admin)
// @Description  Activates or deactivates a user account.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Param        request body AdminUpdateUserStatusRequest true "Update Status Request"
// @Security     BearerAuth
// @Success      200  {object}  api.Response
// @Failure      400  {object}  api.Response{error=api.ErrorInfo}
// @Failure      500  {object}  api.Response{error=api.ErrorInfo}
// @Router       /api/v1/admin/users/{id}/status [patch]
func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.BadRequestResponse(c, "Invalid user ID format")
		return
	}

	var req AdminUpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	v := validator.New()
	req.Validate(v)
	if !v.Valid() {
		api.ValidationErrorResponse(c, validator.NewValidationError("Validation failed", v.Errors))
		return
	}

	if err := h.service.UpdateUserStatus(c.Request.Context(), userID, *req.IsActive); err != nil {
		h.logger.Error(err, logger.Fields{"handler": "UpdateUserStatus", "user_id": userID})
		api.InternalErrorResponse(c, "Failed to update user status")
		return
	}

	api.SuccessResponse(c, 200, "User status updated successfully", nil)
}

// AssignRoleToUser godoc
// @Summary      Assign role to user (Admin)
// @Description  Assigns a new role to a user. This will replace any existing roles.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Param        request body AdminAssignRoleRequest true "Assign Role Request"
// @Security     BearerAuth
// @Success      200  {object}  api.Response
// @Failure      400  {object}  api.Response{error=api.ErrorInfo}
// @Failure      500  {object}  api.Response{error=api.ErrorInfo}
// @Router       /api/v1/admin/users/{id}/assign-role [post]
func (h *AdminHandler) AssignRoleToUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.BadRequestResponse(c, "Invalid user ID format")
		return
	}

	var req AdminAssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	v := validator.New()
	req.Validate(v)
	if !v.Valid() {
		api.ValidationErrorResponse(c, validator.NewValidationError("Validation failed", v.Errors))
		return
	}

	if err := h.service.AssignRole(c.Request.Context(), userID, req.RoleID); err != nil {
		h.logger.Error(err, logger.Fields{"handler": "AssignRoleToUser", "user_id": userID, "role_id": req.RoleID})
		api.InternalErrorResponse(c, "Failed to assign role")
		return
	}

	api.SuccessResponse(c, 200, "Role assigned successfully", nil)
}

// BulkAssignPermissions godoc
// @Summary      Bulk assign permissions (Admin)
// @Description  Assigns one or more permissions to one or more users.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        request body AdminBulkAssignRequest true "Bulk Assign Permissions Request"
// @Security     BearerAuth
// @Success      200  {object}  api.Response
// @Failure      400  {object}  api.Response{error=api.ErrorInfo}
// @Failure      500  {object}  api.Response{error=api.ErrorInfo}
// @Router       /api/v1/admin/users/bulk-assign-permissions [post]
func (h *AdminHandler) BulkAssignPermissions(c *gin.Context) {
	var req AdminBulkAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	v := validator.New()
	req.Validate(v)
	if !v.Valid() {
		api.ValidationErrorResponse(c, validator.NewValidationError("Validation failed", v.Errors))
		return
	}

	if err := h.service.BulkAssignPermissions(c.Request.Context(), req.UserIDs, req.PermissionIDs); err != nil {
		h.logger.Error(err, logger.Fields{"handler": "BulkAssignPermissions"})
		api.InternalErrorResponse(c, "Failed to bulk assign permissions")
		return
	}

	api.SuccessResponse(c, 200, "Permissions assigned successfully", nil)
}
