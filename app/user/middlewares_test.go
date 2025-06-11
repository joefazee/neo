package user

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/joefazee/neo/internal/security"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]string), args.Error(1)
}

type AuthMiddlewareTestSuite struct {
	suite.Suite
	tokenMaker  *security.MockMaker
	authService *MockAuthService
	router      *gin.Engine
}

func (suite *AuthMiddlewareTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
}

func (suite *AuthMiddlewareTestSuite) SetupTest() {
	suite.tokenMaker = &security.MockMaker{}
	suite.authService = &MockAuthService{}
	suite.router = gin.New()

	suite.router.Use(AuthMiddleware(suite.tokenMaker, suite.authService))
	suite.router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
}

func TestAuthMiddleware(t *testing.T) {
	suite.Run(t, new(AuthMiddlewareTestSuite))
}

func (suite *AuthMiddlewareTestSuite) TestMissingAuthHeader() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *AuthMiddlewareTestSuite) TestEmptyAuthHeader() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set("Authorization", "")

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *AuthMiddlewareTestSuite) TestInvalidAuthHeaderFormat_NoBearer() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set("Authorization", "Basic token123")

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *AuthMiddlewareTestSuite) TestInvalidAuthHeaderFormat_OnlyBearer() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer")

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *AuthMiddlewareTestSuite) TestInvalidToken() {
	suite.tokenMaker.On("VerifyToken", "invalid_token").Return(nil, errors.New("invalid token"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer invalid_token")

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)
	suite.tokenMaker.AssertExpectations(suite.T())
}

func (suite *AuthMiddlewareTestSuite) TestAuthServiceError() {
	userID := uuid.New()
	payload := &security.Payload{UserID: userID}

	suite.tokenMaker.On("VerifyToken", "valid_token").Return(payload, nil)
	suite.authService.On("GetUserPermissions", mock.Anything, userID).Return([]string{}, errors.New("service error"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer valid_token")

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusForbidden, w.Code)
	suite.tokenMaker.AssertExpectations(suite.T())
	suite.authService.AssertExpectations(suite.T())
}

func (suite *AuthMiddlewareTestSuite) TestSuccessful() {
	userID := uuid.New()
	payload := &security.Payload{UserID: userID}
	permissions := []string{"read", "write"}

	suite.tokenMaker.On("VerifyToken", "valid_token").Return(payload, nil)
	suite.authService.On("GetUserPermissions", mock.Anything, userID).Return(permissions, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer valid_token")

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
	suite.tokenMaker.AssertExpectations(suite.T())
	suite.authService.AssertExpectations(suite.T())
}

func (suite *AuthMiddlewareTestSuite) TestContextValues() {
	userID := uuid.New()
	payload := &security.Payload{UserID: userID}
	permissions := []string{"admin"}

	suite.tokenMaker.On("VerifyToken", "valid_token").Return(payload, nil)
	suite.authService.On("GetUserPermissions", mock.Anything, userID).Return(permissions, nil)

	suite.router.GET("/context-test", func(c *gin.Context) {
		contextUserID, exists := c.Get("userID")
		assert.True(suite.T(), exists)
		assert.Equal(suite.T(), userID, contextUserID)

		contextPermissions, exists := c.Get("permissions")
		assert.True(suite.T(), exists)
		assert.Equal(suite.T(), permissions, contextPermissions)

		c.JSON(http.StatusOK, gin.H{"userID": contextUserID, "permissions": contextPermissions})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/context-test", http.NoBody)
	req.Header.Set("Authorization", "Bearer valid_token")

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
}
