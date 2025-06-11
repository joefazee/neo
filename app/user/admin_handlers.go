package user

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joefazee/neo/app/api"
	"github.com/joefazee/neo/internal/logger"
	"github.com/joefazee/neo/internal/sanitizer"
	"github.com/joefazee/neo/internal/validator"
	"github.com/joefazee/neo/models"
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

// CreatePermission godoc
// @Summary Create a new permission
// @Description Create a new permission in the system
// @Tags admin-permissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreatePermissionRequest true "Permission creation request"
// @Success 201 {object} api.Response{data=PermissionResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 403 {object} api.Response{error=api.ErrorInfo}
// @Failure 409 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/admin/permissions [post]
func (h *AdminHandler) CreatePermission(c *gin.Context) {
	var req CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	permission, err := h.service.CreatePermission(c.Request.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			api.ConflictResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to create permission")
		return
	}

	api.CreatedResponse(c, "Permission created successfully", permission)
}

// CreateRole godoc
// @Summary Create a new role
// @Description Create a new role in the system
// @Tags admin-roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateRoleRequest true "Role creation request"
// @Success 201 {object} api.Response{data=RoleResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 403 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/admin/roles [post]
func (h *AdminHandler) CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	role, err := h.service.CreateRole(c.Request.Context(), &req)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to create role")
		return
	}

	api.CreatedResponse(c, "Role created successfully", role)
}

// UpdateRole godoc
// @Summary Update an existing role
// @Description Update an existing role's name and description
// @Tags admin-roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Role ID"
// @Param request body UpdateRoleRequest true "Role update request"
// @Success 200 {object} api.Response{data=RoleResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 403 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/admin/roles/{id} [put]
func (h *AdminHandler) UpdateRole(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		api.BadRequestResponse(c, "Invalid role ID format")
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	role, err := h.service.UpdateRole(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "Role")
			return
		}
		api.InternalErrorResponse(c, "Failed to update role")
		return
	}

	api.UpdatedResponse(c, "Role updated successfully", role)
}

// AssignPermissionsToRole godoc
// @Summary Assign permissions to a role
// @Description Assign multiple permissions to a role using permission codes
// @Tags admin-roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Role ID"
// @Param request body AssignPermissionsRequest true "Permissions assignment request"
// @Success 200 {object} api.Response{data=RoleResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 403 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/admin/roles/{id}/permissions [post]
// //nolint: dupl
func (h *AdminHandler) AssignPermissionsToRole(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		api.BadRequestResponse(c, "Invalid role ID format")
		return
	}

	var req AssignPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	role, err := h.service.AssignPermissionsToRole(c.Request.Context(), id, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.NotFoundResponse(c, "Role or one or more permissions")
			return
		}
		api.InternalErrorResponse(c, "Failed to assign permissions to role")
		return
	}

	api.SuccessResponse(c, 200, "Permissions assigned to role successfully", role)
}

// RemovePermissionsFromRole godoc
// @Summary Remove permissions from a role
// @Description Remove multiple permissions from a role using permission codes
// @Tags admin-roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Role ID"
// @Param request body RemovePermissionsRequest true "Permissions removal request"
// @Success 200 {object} api.Response{data=RoleResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 403 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/admin/roles/{id}/permissions [delete]
// //nolint: dupl
func (h *AdminHandler) RemovePermissionsFromRole(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		api.BadRequestResponse(c, "Invalid role ID format")
		return
	}

	var req RemovePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, err.Error())
		return
	}

	role, err := h.service.RemovePermissionsFromRole(c.Request.Context(), id, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.NotFoundResponse(c, "Role or one or more permissions")
			return
		}
		api.InternalErrorResponse(c, "Failed to remove permissions from role")
		return
	}

	api.SuccessResponse(c, 200, "Permissions removed from role successfully", role)
}

// GetUserByID godoc
// @Summary Get user by ID
// @Description Get detailed information about a specific user including roles and permissions
// @Tags admin-users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} api.Response{data=AdminUserResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 403 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/admin/users/{id} [get]
func (h *AdminHandler) GetUserByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		api.BadRequestResponse(c, "Invalid user ID format")
		return
	}

	user, err := h.service.GetUserByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "User")
			return
		}
		api.InternalErrorResponse(c, "Failed to get user")
		return
	}

	api.SuccessResponse(c, 200, "User retrieved successfully", user)
}

// RemoveRoleFromUser godoc
// @Summary Remove role from user
// @Description Remove a specific role from a user
// @Tags admin-users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param role_id path string true "Role ID"
// @Success 200 {object} api.Response{data=AdminUserResponse}
// @Failure 400 {object} api.Response{error=api.ErrorInfo}
// @Failure 401 {object} api.Response{error=api.ErrorInfo}
// @Failure 403 {object} api.Response{error=api.ErrorInfo}
// @Failure 404 {object} api.Response{error=api.ErrorInfo}
// @Failure 500 {object} api.Response{error=api.ErrorInfo}
// @Router /api/v1/admin/users/{id}/roles/{role_id} [delete]
func (h *AdminHandler) RemoveRoleFromUser(c *gin.Context) {
	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		api.BadRequestResponse(c, "Invalid user ID format")
		return
	}

	roleIDParam := c.Param("role_id")
	roleID, err := uuid.Parse(roleIDParam)
	if err != nil {
		api.BadRequestResponse(c, "Invalid role ID format")
		return
	}

	user, err := h.service.RemoveRoleFromUser(c.Request.Context(), userID, roleID)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			api.NotFoundResponse(c, "User")
			return
		}
		if strings.Contains(err.Error(), "does not have this role") {
			api.BadRequestResponse(c, "User does not have this role")
			return
		}
		api.InternalErrorResponse(c, "Failed to remove role from user")
		return
	}

	api.SuccessResponse(c, 200, "Role removed from user successfully", user)
}
