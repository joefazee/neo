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

type MockService struct {
	mock.Mock
}

func (m *MockService) Register(ctx context.Context, req *RegisterUserRequest) (*Response, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Response), args.Error(1)
}

func (m *MockService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LoginResponse), args.Error(1)
}

func (m *MockService) RequestPasswordReset(ctx context.Context, email string) error {
	return m.Called(ctx, email).Error(0)
}

func (m *MockService) ResetPassword(ctx context.Context, token, newPassword string) error {
	return m.Called(ctx, token, newPassword).Error(0)
}

func (m *MockService) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return m.Called(ctx, userID, roleID).Error(0)
}

type UserHandlerTestSuite struct {
	suite.Suite
	handler     *Handler
	service     *MockService
	countryRepo *MockCountryRepo
	sanitizer   *MockSanitizer
	router      *gin.Engine
}

func (suite *UserHandlerTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
}

func (suite *UserHandlerTestSuite) SetupTest() {
	suite.service = &MockService{}
	suite.countryRepo = &MockCountryRepo{}
	suite.sanitizer = &MockSanitizer{}
	suite.handler = NewHandler(suite.service, suite.countryRepo, suite.sanitizer, logger.NewNullLogger())
	suite.router = gin.New()
}

func TestUserHandler(t *testing.T) {
	suite.Run(t, new(UserHandlerTestSuite))
}

func (suite *UserHandlerTestSuite) TestRegister_Success() {
	isActive := true
	country := &models.Country{ID: uuid.New(), Code: "US", IsActive: &isActive}
	response := &Response{ID: uuid.New(), Email: "john@example.com"}

	suite.setupRegisterMocks("John", "Doe", "john@example.com", "US", "+1234567890")
	suite.countryRepo.On("GetByCode", mock.Anything, "US").Return(country, nil)
	suite.service.On("Register", mock.Anything, mock.MatchedBy(func(req *RegisterUserRequest) bool {
		return req.Email == "john@example.com"
	})).Return(response, nil)

	reqBody := RegisterUserRequest{
		FirstName:   "John",
		LastName:    "Doe",
		Email:       "john@example.com",
		CountryCode: "US",
		PhoneNumber: "+1234567890",
		Password:    "password123",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.Register(c)

	suite.Equal(http.StatusCreated, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *UserHandlerTestSuite) TestRegister_BindJSONError() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/register", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.Register(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *UserHandlerTestSuite) TestRegister_ValidationError() {
	suite.setupRegisterMocks("", "Doe", "john@example.com", "US", "+1234567890")
	suite.countryRepo.On("GetByCode", mock.Anything, "US").Return(nil, errors.New("not found"))

	reqBody := RegisterUserRequest{
		FirstName:   "",
		LastName:    "Doe",
		Email:       "john@example.com",
		CountryCode: "US",
		PhoneNumber: "+1234567890",
		Password:    "password123",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.Register(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *UserHandlerTestSuite) TestRegister_ServiceError() {
	isActive := true
	country := &models.Country{ID: uuid.New(), Code: "US", IsActive: &isActive}

	suite.setupRegisterMocks("John", "Doe", "john@example.com", "US", "+1234567890")
	suite.countryRepo.On("GetByCode", mock.Anything, "US").Return(country, nil)
	suite.service.On("Register", mock.Anything, mock.Anything).Return(nil, errors.New("service error"))

	reqBody := RegisterUserRequest{
		FirstName:   "John",
		LastName:    "Doe",
		Email:       "john@example.com",
		CountryCode: "US",
		PhoneNumber: "+1234567890",
		Password:    "password123",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.Register(c)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

func (suite *UserHandlerTestSuite) TestLogin_Success() {
	response := &LoginResponse{AccessToken: "token123", User: Response{ID: uuid.New()}}

	suite.sanitizer.On("StripHTML", "john@example.com").Return("john@example.com")
	suite.service.On("Login", mock.Anything, mock.MatchedBy(func(req *LoginRequest) bool {
		return req.Identity == "john@example.com"
	})).Return(response, nil)

	reqBody := LoginRequest{
		Identity: "john@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.Login(c)

	suite.Equal(http.StatusOK, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *UserHandlerTestSuite) TestLogin_BindJSONError() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.Login(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *UserHandlerTestSuite) TestLogin_ValidationError() {
	suite.sanitizer.On("StripHTML", "").Return("")

	reqBody := LoginRequest{
		Identity: "",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.Login(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *UserHandlerTestSuite) TestLogin_ServiceError() {
	suite.sanitizer.On("StripHTML", "john@example.com").Return("john@example.com")
	suite.service.On("Login", mock.Anything, mock.Anything).Return(nil, errors.New("invalid credentials"))

	reqBody := LoginRequest{
		Identity: "john@example.com",
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.Login(c)

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *UserHandlerTestSuite) TestRequestPasswordReset_Success() {
	suite.service.On("RequestPasswordReset", mock.Anything, "john@example.com").Return(nil)

	reqBody := PasswordResetRequest{Email: "john@example.com"}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/password-reset", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.RequestPasswordReset(c)

	suite.Equal(http.StatusOK, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *UserHandlerTestSuite) TestRequestPasswordReset_BindJSONError() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/password-reset", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.RequestPasswordReset(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *UserHandlerTestSuite) TestRequestPasswordReset_ServiceError() {
	suite.service.On("RequestPasswordReset", mock.Anything, "john@example.com").Return(errors.New("service error"))

	reqBody := PasswordResetRequest{Email: "john@example.com"}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/password-reset", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.RequestPasswordReset(c)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

func (suite *UserHandlerTestSuite) TestResetPassword_Success() {
	suite.service.On("ResetPassword", mock.Anything, "token123", "newpassword123").Return(nil)

	reqBody := SetNewPasswordRequest{
		Token:       "token123",
		NewPassword: "newpassword123",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/reset-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.ResetPassword(c)

	suite.Equal(http.StatusOK, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *UserHandlerTestSuite) TestResetPassword_BindJSONError() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/reset-password", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.ResetPassword(c)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *UserHandlerTestSuite) TestResetPassword_ServiceError() {
	suite.service.On("ResetPassword", mock.Anything, "token123", "newpassword123").Return(errors.New("invalid token"))

	reqBody := SetNewPasswordRequest{
		Token:       "token123",
		NewPassword: "newpassword123",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/reset-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.ResetPassword(c)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

func (suite *UserHandlerTestSuite) TestGetProfile_Success() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/profile", http.NoBody)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.GetProfile(c)

	suite.Equal(http.StatusOK, w.Code)
}

func (suite *UserHandlerTestSuite) setupRegisterMocks(firstName, lastName, email, _, phone string) {
	suite.sanitizer.On("StripHTML", firstName).Return(firstName)
	suite.sanitizer.On("StripHTML", lastName).Return(lastName)
	suite.sanitizer.On("StripHTML", email).Return(email)
	suite.sanitizer.On("StripHTML", phone).Return(phone)
}
