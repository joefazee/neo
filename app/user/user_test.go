package user

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/joefazee/neo/internal/security"
	"github.com/joefazee/neo/models"
)

type MiddlewareTestSuite struct {
	suite.Suite
	tokenMaker *security.MockMaker
	repo       *MockRepo
	router     *gin.Engine
}

func (suite *MiddlewareTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
}

func (suite *MiddlewareTestSuite) SetupTest() {
	suite.tokenMaker = &security.MockMaker{}
	suite.repo = &MockRepo{}
	suite.router = gin.New()
}

func TestMiddleware(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}

func (suite *MiddlewareTestSuite) TestContextSetUser() {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	user := &models.User{ID: uuid.New()}

	ContextSetUser(c, user)

	result, exists := c.Get(ContextUser)
	suite.True(exists)
	suite.Equal(user, result)
}

func (suite *MiddlewareTestSuite) TestContextSetToken() {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	payload := &security.Payload{UserID: uuid.New()}

	ContextSetToken(c, payload)

	result, exists := c.Get(ContextToken)
	suite.True(exists)
	suite.Equal(payload, result)
}

func (suite *MiddlewareTestSuite) TestContextGetUser() {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	user := &models.User{ID: uuid.New()}
	c.Set(ContextUser, user)

	result := ContextGetUser(c)

	suite.Equal(user, result)
}

func (suite *MiddlewareTestSuite) TestContextGetUser_Panic() {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	suite.Panics(func() {
		ContextGetUser(c)
	})
}

func (suite *MiddlewareTestSuite) TestContextGetToken() {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	payload := &security.Payload{UserID: uuid.New()}
	c.Set(ContextToken, payload)

	result := ContextGetToken(c)

	suite.Equal(payload, result)
}

func (suite *MiddlewareTestSuite) TestContextGetToken_Panic() {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	suite.Panics(func() {
		ContextGetToken(c)
	})
}

func (suite *MiddlewareTestSuite) TestApplyAuthentication_NoHeader() {
	suite.router.Use(ApplyAuthentication(suite.tokenMaker, suite.repo))
	suite.router.GET("/test", func(c *gin.Context) {
		user := ContextGetUser(c)
		suite.Equal(AnonymousUser, user)
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
	suite.Contains(w.Header().Get("Vary"), AuthorizationHeaderKey)
}

func (suite *MiddlewareTestSuite) TestApplyAuthentication_InvalidHeaderFormat_NotTwoParts() {
	suite.router.Use(ApplyAuthentication(suite.tokenMaker, suite.repo))
	suite.router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set(AuthorizationHeaderKey, "Bearer")
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *MiddlewareTestSuite) TestApplyAuthentication_InvalidHeaderFormat_NotBearer() {
	suite.router.Use(ApplyAuthentication(suite.tokenMaker, suite.repo))
	suite.router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set(AuthorizationHeaderKey, "Basic token123")
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *MiddlewareTestSuite) TestApplyAuthentication_ExpiredToken() {
	suite.tokenMaker.On("VerifyToken", "expired_token").Return(nil, errors.New("token expired"))

	suite.router.Use(ApplyAuthentication(suite.tokenMaker, suite.repo))
	suite.router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set(AuthorizationHeaderKey, "Bearer expired_token")
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *MiddlewareTestSuite) TestApplyAuthentication_OtherTokenError() {
	userID := uuid.New()
	payload := &security.Payload{UserID: userID}
	user := &models.User{ID: userID}

	suite.tokenMaker.On("VerifyToken", "invalid_token").Return(payload, errors.New("invalid signature"))
	suite.repo.On("GetByEmail", mock.Anything, userID.String()).Return(user, nil)

	suite.router.Use(ApplyAuthentication(suite.tokenMaker, suite.repo))
	suite.router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set(AuthorizationHeaderKey, "Bearer invalid_token")
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
}

func (suite *MiddlewareTestSuite) TestApplyAuthentication_RefreshTokenOnInvalidEndpoint() {
	payload := &security.Payload{
		UserID: uuid.New(),
		Scope:  security.TokenScopeRefresh,
	}

	suite.tokenMaker.On("VerifyToken", "refresh_token").Return(payload, nil)

	suite.router.Use(ApplyAuthentication(suite.tokenMaker, suite.repo))
	suite.router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set(AuthorizationHeaderKey, "Bearer refresh_token")
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *MiddlewareTestSuite) TestApplyAuthentication_RefreshTokenOnValidEndpoint() {
	userID := uuid.New()
	payload := &security.Payload{
		UserID: userID,
		Scope:  security.TokenScopeRefresh,
	}
	user := &models.User{ID: userID}

	suite.tokenMaker.On("VerifyToken", "refresh_token").Return(payload, nil)
	suite.repo.On("GetByEmail", mock.Anything, userID.String()).Return(user, nil)

	suite.router.Use(ApplyAuthentication(suite.tokenMaker, suite.repo))
	suite.router.GET("/api/v1/users/refresh-token", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/refresh-token", http.NoBody)
	req.Header.Set(AuthorizationHeaderKey, "Bearer refresh_token")
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
}

func (suite *MiddlewareTestSuite) TestApplyAuthentication_UserNotFound() {
	userID := uuid.New()
	payload := &security.Payload{UserID: userID}

	suite.tokenMaker.On("VerifyToken", "valid_token").Return(payload, nil)
	suite.repo.On("GetByEmail", mock.Anything, userID.String()).Return(nil, errors.New("user not found"))

	suite.router.Use(ApplyAuthentication(suite.tokenMaker, suite.repo))
	suite.router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set(AuthorizationHeaderKey, "Bearer valid_token")
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *MiddlewareTestSuite) TestApplyAuthentication_Success() {
	userID := uuid.New()
	payload := &security.Payload{UserID: userID}
	user := &models.User{ID: userID}

	suite.tokenMaker.On("VerifyToken", "valid_token").Return(payload, nil)
	suite.repo.On("GetByEmail", mock.Anything, userID.String()).Return(user, nil)

	suite.router.Use(ApplyAuthentication(suite.tokenMaker, suite.repo))
	suite.router.GET("/test", func(c *gin.Context) {
		contextUser := ContextGetUser(c)
		contextToken := ContextGetToken(c)
		suite.Equal(user, contextUser)
		suite.Equal(payload, contextToken)
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set(AuthorizationHeaderKey, "Bearer valid_token")
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
}

func (suite *MiddlewareTestSuite) TestContains_Found() {
	endpoints := []string{"/api/v1/users/refresh-token", "/api/v1/auth/refresh"}
	result := contains(endpoints, "/api/v1/users/refresh-token")
	suite.True(result)
}

func (suite *MiddlewareTestSuite) TestContains_NotFound() {
	endpoints := []string{"/api/v1/users/refresh-token"}
	result := contains(endpoints, "/api/v1/users/login")
	suite.False(result)
}

func (suite *MiddlewareTestSuite) TestActivatedUserRequired_InactiveUser() {
	isActive := false
	user := &models.User{ID: uuid.New(), IsActive: &isActive}

	suite.router.Use(func(c *gin.Context) {
		ContextSetUser(c, user)
		c.Next()
	})
	suite.router.Use(ActivatedUserRequired())
	suite.router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *MiddlewareTestSuite) TestActivatedUserRequired_ActiveUser() {
	isActive := true
	user := &models.User{ID: uuid.New(), IsActive: &isActive}

	suite.router.Use(func(c *gin.Context) {
		ContextSetUser(c, user)
		c.Next()
	})
	suite.router.Use(ActivatedUserRequired())
	suite.router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
}

func (suite *MiddlewareTestSuite) TestAuthenticatedUseRequired_AnonymousUser() {
	suite.router.Use(func(c *gin.Context) {
		ContextSetUser(c, AnonymousUser)
		c.Next()
	})
	suite.router.Use(AuthenticatedUseRequired())
	suite.router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *MiddlewareTestSuite) TestAuthenticatedUseRequired_RealUser() {
	user := &models.User{ID: uuid.New()}

	suite.router.Use(func(c *gin.Context) {
		ContextSetUser(c, user)
		c.Next()
	})
	suite.router.Use(AuthenticatedUseRequired())
	suite.router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
}

func (suite *MiddlewareTestSuite) TestRequireActivatedUser_InactiveUser() {
	isActive := false
	user := &models.User{ID: uuid.New(), IsActive: &isActive}

	suite.router.Use(func(c *gin.Context) {
		ContextSetUser(c, user)
		c.Next()
	})
	suite.router.Use(RequireActivatedUser())
	suite.router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *MiddlewareTestSuite) TestRequireActivatedUser_ActiveUser() {
	isActive := true
	user := &models.User{ID: uuid.New(), IsActive: &isActive}

	suite.router.Use(func(c *gin.Context) {
		ContextSetUser(c, user)
		c.Next()
	})
	suite.router.Use(RequireActivatedUser())
	suite.router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
}
