package user

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joefazee/neo/app/api"
	"github.com/joefazee/neo/internal/security"
	"github.com/joefazee/neo/models"
)

const (
	AuthorizedMerchantAPIKey = "APIKey"
	AuthorizationHeaderKey   = "Authorization"
	AuthorizationTypeBearer  = "Bearer"
	PinHeaderKey             = "X-Pin"
)

var (
	validRefreshTokenEndpoints = []string{
		"/api/v1/users/refresh-token",
	}
)

const (
	ContextUser           = "context_user"
	ContextToken          = "context_token"
	ContextSystemSettings = "context_system_settings"
)

var AnonymousUser = &models.User{
	ID: uuid.Nil,
}

// ContextSetUser sets the user in the context
func ContextSetUser(c *gin.Context, user *models.User) *gin.Context {
	c.Set(ContextUser, user)
	return c
}

// ContextSetToken sets the user in the context
func ContextSetToken(c *gin.Context, payload *security.Payload) *gin.Context {
	c.Set(ContextToken, payload)
	return c
}

// ContextGetUser gets the user from the context
func ContextGetUser(c *gin.Context) *models.User {
	user, ok := c.Get(ContextUser)
	if !ok {
		panic("missing user value in context")
	}
	return user.(*models.User)
}

// ContextGetToken gets the user from the context
func ContextGetToken(c *gin.Context) *security.Payload {
	token, ok := c.Get(ContextToken)
	if !ok {
		panic("missing token value in context")
	}
	return token.(*security.Payload)
}

// ApplyAuthentication is a middleware that checks for the authorization header
// the function does not check if the user is activated or not
func ApplyAuthentication(tokenMaker security.Maker, repo Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Vary", AuthorizationHeaderKey)

		authHeader := c.GetHeader(AuthorizationHeaderKey)
		if authHeader == "" {
			ContextSetUser(c, AnonymousUser)
			c.Next()
			return
		}

		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != AuthorizationTypeBearer {
			api.UnauthorizedResponse(c)
			return
		}

		token := headerParts[1]
		payload, err := tokenMaker.VerifyToken(token)

		if err != nil {
			if strings.Contains(err.Error(), "expired") {
				api.UnauthorizedResponse(c)
				return
			}
		}

		if payload.Scope == security.TokenScopeRefresh && !contains(validRefreshTokenEndpoints, c.Request.URL.Path) {
			api.UnauthorizedResponse(c)
			return
		}

		user, err := repo.GetByEmail(c, payload.UserID.String())
		if err != nil {
			api.UnauthorizedResponse(c)
			return
		}

		ContextSetUser(c, user)
		ContextSetToken(c, payload)

		c.Next()
	}
}

func contains(endpoints []string, path string) bool {
	for i := range endpoints {
		endpoint := endpoints[i]
		if endpoint == path {
			return true
		}
	}
	return false
}

func ActivatedUserRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := ContextGetUser(c)
		if !*user.IsActive {
			api.UnauthorizedResponse(c)
			return
		}
		c.Next()
	}
}

// AuthenticatedUseRequired is a middleware that checks if the user is authenticated
// if the user is Anonymous, it aborts the request
func AuthenticatedUseRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := ContextGetUser(c)
		if user.IsAnonymous() {
			api.UnauthorizedResponse(c)
			return
		}
		c.Next()
	}
}

// RequireActivatedUser is a middleware that checks if the user is activated
// if the user is not activated, it aborts the request
func RequireActivatedUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := ContextGetUser(c)
		if !*user.IsActive {
			api.UnauthorizedResponse(c)
			return
		}
		c.Next()
	}
}
