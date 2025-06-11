package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/joefazee/neo/internal/logger"
	"github.com/joefazee/neo/models"
)

type MockAdminService struct {
	mock.Mock
}

func (m *MockAdminService) GetUsers(ctx context.Context, filters *AdminUserFilters) ([]AdminUserResponse, int64, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]AdminUserResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockAdminService) UpdateUserStatus(ctx context.Context, userID uuid.UUID, isActive bool) error {
	return m.Called(ctx, userID, isActive).Error(0)
}

func (m *MockAdminService) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return m.Called(ctx, userID, roleID).Error(0)
}

func (m *MockAdminService) BulkAssignPermissions(ctx context.Context, userIDs, permissionIDs []uuid.UUID) error {
	return m.Called(ctx, userIDs, permissionIDs).Error(0)
}

func (m *MockAdminService) CreatePermission(ctx context.Context, req *CreatePermissionRequest) (*PermissionResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PermissionResponse), args.Error(1)
}

func (m *MockAdminService) CreateRole(ctx context.Context, req *CreateRoleRequest) (*RoleResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RoleResponse), args.Error(1)
}

func (m *MockAdminService) UpdateRole(ctx context.Context, id uuid.UUID, req *UpdateRoleRequest) (*RoleResponse, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RoleResponse), args.Error(1)
}

func (m *MockAdminService) AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, req *AssignPermissionsRequest) (*RoleResponse, error) {
	args := m.Called(ctx, roleID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RoleResponse), args.Error(1)
}

func (m *MockAdminService) RemovePermissionsFromRole(ctx context.Context, roleID uuid.UUID, req *RemovePermissionsRequest) (*RoleResponse, error) {
	args := m.Called(ctx, roleID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RoleResponse), args.Error(1)
}

func (m *MockAdminService) GetUserByID(ctx context.Context, id uuid.UUID) (*AdminUserResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AdminUserResponse), args.Error(1)
}

func (m *MockAdminService) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) (*AdminUserResponse, error) {
	args := m.Called(ctx, userID, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AdminUserResponse), args.Error(1)
}

type MockSanitizer struct {
	mock.Mock
}

func (m *MockSanitizer) StripHTML(input string) string {
	return m.Called(input).String(0)
}

type AdminHandlerTestSuite struct {
	suite.Suite
	handler   *AdminHandler
	service   *MockAdminService
	sanitizer *MockSanitizer
	router    *gin.Engine
}

func (suite *AdminHandlerTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
}

func (suite *AdminHandlerTestSuite) SetupTest() {
	suite.service = &MockAdminService{}
	suite.sanitizer = &MockSanitizer{}
	suite.handler = NewAdminHandler(suite.service, suite.sanitizer, logger.NewNullLogger())
	suite.router = gin.New()
}

func TestAdminHandler(t *testing.T) {
	suite.Run(t, new(AdminHandlerTestSuite))
}

func (suite *AdminHandlerTestSuite) TestGetUsers_Success() {
	users := []AdminUserResponse{{ID: uuid.New(), Email: "test@example.com"}}
	suite.sanitizer.On("StripHTML", "").Return("")
	suite.service.On("GetUsers", mock.Anything, mock.MatchedBy(func(f *AdminUserFilters) bool {
		return f.Page == 1 && f.PerPage == 20
	})).Return(users, int64(1), nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users?page=1&per_page=20", http.NoBody)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.GetUsers(c)

	suite.Equal(http.StatusOK, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *AdminHandlerTestSuite) TestGetUsers_ValidationError() {
	suite.sanitizer.On("StripHTML", "").Return("")
	suite.sanitizer.On("StripHTML", "invalid").Return("invalid")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users?status=invalid", http.NoBody)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.GetUsers(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestGetUsers_ServiceError() {
	suite.sanitizer.On("StripHTML", "").Return("")
	suite.service.On("GetUsers", mock.Anything, mock.Anything).Return([]AdminUserResponse{}, int64(0), errors.New("service error"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users", http.NoBody)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.GetUsers(c)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

func (suite *AdminHandlerTestSuite) TestUpdateUserStatus_Success() {
	userID := uuid.New()
	reqBody := AdminUpdateUserStatusRequest{IsActive: ptrBool(false)}
	body, _ := json.Marshal(reqBody)

	suite.service.On("UpdateUserStatus", mock.Anything, userID, false).Return(nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/users/"+userID.String()+"/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	suite.handler.UpdateUserStatus(c)

	suite.Equal(http.StatusOK, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *AdminHandlerTestSuite) TestUpdateUserStatus_InvalidID() {
	reqBody := AdminUpdateUserStatusRequest{IsActive: ptrBool(false)}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/users/invalid/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	suite.handler.UpdateUserStatus(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestUpdateUserStatus_ValidationError() {
	userID := uuid.New()
	reqBody := map[string]interface{}{"is_active": nil}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/users/"+userID.String()+"/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	suite.handler.UpdateUserStatus(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestAssignRoleToUser_Success() {
	userID := uuid.New()
	roleID := uuid.New()
	reqBody := AdminAssignRoleRequest{RoleID: roleID}
	body, _ := json.Marshal(reqBody)

	suite.service.On("AssignRole", mock.Anything, userID, roleID).Return(nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/users/"+userID.String()+"/assign-role", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	suite.handler.AssignRoleToUser(c)

	suite.Equal(http.StatusOK, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *AdminHandlerTestSuite) TestBulkAssignPermissions_Success() {
	userIDs := []uuid.UUID{uuid.New(), uuid.New()}
	permIDs := []uuid.UUID{uuid.New()}
	reqBody := AdminBulkAssignRequest{UserIDs: userIDs, PermissionIDs: permIDs}
	body, _ := json.Marshal(reqBody)

	suite.service.On("BulkAssignPermissions", mock.Anything, userIDs, permIDs).Return(nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/users/bulk-assign-permissions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.BulkAssignPermissions(c)

	suite.Equal(http.StatusOK, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *AdminHandlerTestSuite) TestCreatePermission_Success() {
	reqBody := CreatePermissionRequest{Name: "test_permission", Description: "Test"}
	body, _ := json.Marshal(reqBody)
	response := &PermissionResponse{ID: uuid.New(), Name: "test_permission"}

	suite.service.On("CreatePermission", mock.Anything, &reqBody).Return(response, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/permissions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.CreatePermission(c)

	suite.Equal(http.StatusCreated, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *AdminHandlerTestSuite) TestCreatePermission_Conflict() {
	reqBody := CreatePermissionRequest{Name: "existing_permission", Description: "Test"}
	body, _ := json.Marshal(reqBody)

	suite.service.On("CreatePermission", mock.Anything, &reqBody).Return(nil, errors.New("permission with this name already exists"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/permissions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.CreatePermission(c)

	suite.Equal(http.StatusConflict, w.Code)
}

func (suite *AdminHandlerTestSuite) TestCreateRole_Success() {
	reqBody := CreateRoleRequest{Name: "test_role", Description: "Test"}
	body, _ := json.Marshal(reqBody)
	response := &RoleResponse{ID: uuid.New(), Name: "test_role"}

	suite.service.On("CreateRole", mock.Anything, &reqBody).Return(response, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/roles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.CreateRole(c)

	suite.Equal(http.StatusCreated, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *AdminHandlerTestSuite) TestUpdateRole_Success() {
	roleID := uuid.New()
	name := "updated_role"
	reqBody := UpdateRoleRequest{Name: &name}
	body, _ := json.Marshal(reqBody)
	response := &RoleResponse{ID: roleID, Name: "updated_role"}

	suite.service.On("UpdateRole", mock.Anything, roleID, &reqBody).Return(response, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/roles/"+roleID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: roleID.String()}}

	suite.handler.UpdateRole(c)

	suite.Equal(http.StatusOK, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *AdminHandlerTestSuite) TestUpdateRole_NotFound() {
	roleID := uuid.New()
	name := "updated_role"
	reqBody := UpdateRoleRequest{Name: &name}
	body, _ := json.Marshal(reqBody)

	suite.service.On("UpdateRole", mock.Anything, roleID, &reqBody).Return(nil, models.ErrRecordNotFound)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/roles/"+roleID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: roleID.String()}}

	suite.handler.UpdateRole(c)

	suite.Equal(http.StatusNotFound, w.Code)
}

func (suite *AdminHandlerTestSuite) TestAssignPermissionsToRole_Success() {
	roleID := uuid.New()
	reqBody := AssignPermissionsRequest{PermissionCodes: []string{"read", "write"}}
	body, _ := json.Marshal(reqBody)
	response := &RoleResponse{ID: roleID, Name: "test_role"}

	suite.service.On("AssignPermissionsToRole", mock.Anything, roleID, &reqBody).Return(response, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/roles/"+roleID.String()+"/permissions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: roleID.String()}}

	suite.handler.AssignPermissionsToRole(c)

	suite.Equal(http.StatusOK, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *AdminHandlerTestSuite) TestAssignPermissionsToRole_NotFound() {
	roleID := uuid.New()
	reqBody := AssignPermissionsRequest{PermissionCodes: []string{"read"}}
	body, _ := json.Marshal(reqBody)

	suite.service.On("AssignPermissionsToRole", mock.Anything, roleID, &reqBody).Return(nil, errors.New("role not found"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/roles/"+roleID.String()+"/permissions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: roleID.String()}}

	suite.handler.AssignPermissionsToRole(c)

	suite.Equal(http.StatusNotFound, w.Code)
}

func (suite *AdminHandlerTestSuite) TestRemovePermissionsFromRole_Success() {
	roleID := uuid.New()
	reqBody := RemovePermissionsRequest{PermissionCodes: []string{"write"}}
	body, _ := json.Marshal(reqBody)
	response := &RoleResponse{ID: roleID, Name: "test_role"}

	suite.service.On("RemovePermissionsFromRole", mock.Anything, roleID, &reqBody).Return(response, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/roles/"+roleID.String()+"/permissions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: roleID.String()}}

	suite.handler.RemovePermissionsFromRole(c)

	suite.Equal(http.StatusOK, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *AdminHandlerTestSuite) TestGetUserByID_Success() {
	userID := uuid.New()
	response := &AdminUserResponse{ID: userID, Email: "test@example.com"}

	suite.service.On("GetUserByID", mock.Anything, userID).Return(response, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/"+userID.String(), http.NoBody)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	suite.handler.GetUserByID(c)

	suite.Equal(http.StatusOK, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *AdminHandlerTestSuite) TestGetUserByID_NotFound() {
	userID := uuid.New()

	suite.service.On("GetUserByID", mock.Anything, userID).Return(nil, models.ErrRecordNotFound)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/"+userID.String(), http.NoBody)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	suite.handler.GetUserByID(c)

	suite.Equal(http.StatusNotFound, w.Code)
}

func (suite *AdminHandlerTestSuite) TestRemoveRoleFromUser_Success() {
	userID := uuid.New()
	roleID := uuid.New()
	response := &AdminUserResponse{ID: userID, Email: "test@example.com"}

	suite.service.On("RemoveRoleFromUser", mock.Anything, userID, roleID).Return(response, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/users/"+userID.String()+"/roles/"+roleID.String(), http.NoBody)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{
		{Key: "id", Value: userID.String()},
		{Key: "role_id", Value: roleID.String()},
	}

	suite.handler.RemoveRoleFromUser(c)

	suite.Equal(http.StatusOK, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *AdminHandlerTestSuite) TestRemoveRoleFromUser_UserNotFound() {
	userID := uuid.New()
	roleID := uuid.New()

	suite.service.On("RemoveRoleFromUser", mock.Anything, userID, roleID).Return(nil, models.ErrRecordNotFound)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/users/"+userID.String()+"/roles/"+roleID.String(), http.NoBody)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{
		{Key: "id", Value: userID.String()},
		{Key: "role_id", Value: roleID.String()},
	}

	suite.handler.RemoveRoleFromUser(c)

	suite.Equal(http.StatusNotFound, w.Code)
}

func (suite *AdminHandlerTestSuite) TestRemoveRoleFromUser_UserDoesNotHaveRole() {
	userID := uuid.New()
	roleID := uuid.New()

	suite.service.On("RemoveRoleFromUser", mock.Anything, userID, roleID).Return(nil, errors.New("user does not have this role"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/users/"+userID.String()+"/roles/"+roleID.String(), http.NoBody)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{
		{Key: "id", Value: userID.String()},
		{Key: "role_id", Value: roleID.String()},
	}

	suite.handler.RemoveRoleFromUser(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestInvalidJSON() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/permissions", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.CreatePermission(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestGetUsers_BindQueryError() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users?page=invalid", http.NoBody)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.GetUsers(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestUpdateUserStatus_BindJSONError() {
	userID := uuid.New()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/users/"+userID.String()+"/status", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	suite.handler.UpdateUserStatus(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestUpdateUserStatus_ServiceError() {
	userID := uuid.New()
	reqBody := AdminUpdateUserStatusRequest{IsActive: ptrBool(false)}
	body, _ := json.Marshal(reqBody)

	suite.service.On("UpdateUserStatus", mock.Anything, userID, false).Return(errors.New("service error"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/users/"+userID.String()+"/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	suite.handler.UpdateUserStatus(c)

	suite.Equal(http.StatusInternalServerError, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *AdminHandlerTestSuite) TestAssignRoleToUser_InvalidUserID() {
	reqBody := AdminAssignRoleRequest{RoleID: uuid.New()}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/users/invalid/assign-role", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	suite.handler.AssignRoleToUser(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestAssignRoleToUser_BindJSONError() {
	userID := uuid.New()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/users/"+userID.String()+"/assign-role", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	suite.handler.AssignRoleToUser(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestAssignRoleToUser_ValidationError() {
	userID := uuid.New()
	reqBody := AdminAssignRoleRequest{RoleID: uuid.Nil}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/users/"+userID.String()+"/assign-role", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	suite.handler.AssignRoleToUser(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestAssignRoleToUser_ServiceError() {
	userID := uuid.New()
	roleID := uuid.New()
	reqBody := AdminAssignRoleRequest{RoleID: roleID}
	body, _ := json.Marshal(reqBody)

	suite.service.On("AssignRole", mock.Anything, userID, roleID).Return(errors.New("service error"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/users/"+userID.String()+"/assign-role", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	suite.handler.AssignRoleToUser(c)

	suite.Equal(http.StatusInternalServerError, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *AdminHandlerTestSuite) TestBulkAssignPermissions_BindJSONError() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/users/bulk-assign-permissions", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.BulkAssignPermissions(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestBulkAssignPermissions_ValidationError() {
	reqBody := AdminBulkAssignRequest{UserIDs: []uuid.UUID{}, PermissionIDs: []uuid.UUID{}}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/users/bulk-assign-permissions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.BulkAssignPermissions(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestBulkAssignPermissions_ServiceError() {
	userIDs := []uuid.UUID{uuid.New()}
	permIDs := []uuid.UUID{uuid.New()}
	reqBody := AdminBulkAssignRequest{UserIDs: userIDs, PermissionIDs: permIDs}
	body, _ := json.Marshal(reqBody)

	suite.service.On("BulkAssignPermissions", mock.Anything, userIDs, permIDs).Return(errors.New("service error"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/users/bulk-assign-permissions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.BulkAssignPermissions(c)

	suite.Equal(http.StatusInternalServerError, w.Code)
	suite.service.AssertExpectations(suite.T())
}

// CreatePermission tests
func (suite *AdminHandlerTestSuite) TestCreatePermission_ServiceError() {
	reqBody := CreatePermissionRequest{Name: "test_permission", Description: "Test"}
	body, _ := json.Marshal(reqBody)

	suite.service.On("CreatePermission", mock.Anything, &reqBody).Return(nil, errors.New("database error"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/permissions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.CreatePermission(c)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

// CreateRole tests
func (suite *AdminHandlerTestSuite) TestCreateRole_BindJSONError() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/roles", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.CreateRole(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestCreateRole_ServiceError() {
	reqBody := CreateRoleRequest{Name: "test_role", Description: "Test"}
	body, _ := json.Marshal(reqBody)

	suite.service.On("CreateRole", mock.Anything, &reqBody).Return(nil, errors.New("database error"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/roles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.CreateRole(c)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

// UpdateRole tests
func (suite *AdminHandlerTestSuite) TestUpdateRole_InvalidID() {
	name := "updated_role"
	reqBody := UpdateRoleRequest{Name: &name}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/roles/invalid", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	suite.handler.UpdateRole(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestUpdateRole_BindJSONError() {
	roleID := uuid.New()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/roles/"+roleID.String(), bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: roleID.String()}}

	suite.handler.UpdateRole(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestUpdateRole_ServiceError() {
	roleID := uuid.New()
	name := "updated_role"
	reqBody := UpdateRoleRequest{Name: &name}
	body, _ := json.Marshal(reqBody)

	suite.service.On("UpdateRole", mock.Anything, roleID, &reqBody).Return(nil, errors.New("database error"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/roles/"+roleID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: roleID.String()}}

	suite.handler.UpdateRole(c)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

// AssignPermissionsToRole tests
func (suite *AdminHandlerTestSuite) TestAssignPermissionsToRole_InvalidID() {
	reqBody := AssignPermissionsRequest{PermissionCodes: []string{"read"}}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/roles/invalid/permissions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	suite.handler.AssignPermissionsToRole(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestAssignPermissionsToRole_BindJSONError() {
	roleID := uuid.New()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/roles/"+roleID.String()+"/permissions", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: roleID.String()}}

	suite.handler.AssignPermissionsToRole(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestAssignPermissionsToRole_ServiceError() {
	roleID := uuid.New()
	reqBody := AssignPermissionsRequest{PermissionCodes: []string{"read"}}
	body, _ := json.Marshal(reqBody)

	suite.service.On("AssignPermissionsToRole", mock.Anything, roleID, &reqBody).Return(nil, errors.New("database error"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/roles/"+roleID.String()+"/permissions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: roleID.String()}}

	suite.handler.AssignPermissionsToRole(c)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

// RemovePermissionsFromRole tests
func (suite *AdminHandlerTestSuite) TestRemovePermissionsFromRole_InvalidID() {
	reqBody := RemovePermissionsRequest{PermissionCodes: []string{"write"}}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/roles/invalid/permissions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	suite.handler.RemovePermissionsFromRole(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestRemovePermissionsFromRole_BindJSONError() {
	roleID := uuid.New()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/roles/"+roleID.String()+"/permissions", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: roleID.String()}}

	suite.handler.RemovePermissionsFromRole(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestRemovePermissionsFromRole_ServiceError() {
	roleID := uuid.New()
	reqBody := RemovePermissionsRequest{PermissionCodes: []string{"write"}}
	body, _ := json.Marshal(reqBody)

	suite.service.On("RemovePermissionsFromRole", mock.Anything, roleID, &reqBody).Return(nil, errors.New("database error"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/roles/"+roleID.String()+"/permissions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: roleID.String()}}

	suite.handler.RemovePermissionsFromRole(c)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

// GetUserByID tests
func (suite *AdminHandlerTestSuite) TestGetUserByID_InvalidID() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/invalid", http.NoBody)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	suite.handler.GetUserByID(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestGetUserByID_ServiceError() {
	userID := uuid.New()

	suite.service.On("GetUserByID", mock.Anything, userID).Return(nil, errors.New("database error"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/"+userID.String(), http.NoBody)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	suite.handler.GetUserByID(c)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

// RemoveRoleFromUser tests
func (suite *AdminHandlerTestSuite) TestRemoveRoleFromUser_InvalidUserID() {
	roleID := uuid.New()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/users/invalid/roles/"+roleID.String(), http.NoBody)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{
		{Key: "id", Value: "invalid"},
		{Key: "role_id", Value: roleID.String()},
	}

	suite.handler.RemoveRoleFromUser(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestRemoveRoleFromUser_InvalidRoleID() {
	userID := uuid.New()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/users/"+userID.String()+"/roles/invalid", http.NoBody)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{
		{Key: "id", Value: userID.String()},
		{Key: "role_id", Value: "invalid"},
	}

	suite.handler.RemoveRoleFromUser(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *AdminHandlerTestSuite) TestRemoveRoleFromUser_ServiceError() {
	userID := uuid.New()
	roleID := uuid.New()

	suite.service.On("RemoveRoleFromUser", mock.Anything, userID, roleID).Return(nil, errors.New("database error"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/users/"+userID.String()+"/roles/"+roleID.String(), http.NoBody)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{
		{Key: "id", Value: userID.String()},
		{Key: "role_id", Value: roleID.String()},
	}

	suite.handler.RemoveRoleFromUser(c)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

func (suite *AdminHandlerTestSuite) TestRemovePermissionsFromRole_NotFoundError() {
	roleID := uuid.New()
	reqBody := RemovePermissionsRequest{PermissionCodes: []string{"write"}}
	body, _ := json.Marshal(reqBody)

	suite.service.On("RemovePermissionsFromRole", mock.Anything, roleID, &reqBody).Return(nil, errors.New("permission not found"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/roles/"+roleID.String()+"/permissions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: roleID.String()}}

	suite.handler.RemovePermissionsFromRole(c)

	suite.Equal(http.StatusNotFound, w.Code)
}
