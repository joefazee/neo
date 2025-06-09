package user

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joefazee/neo/app/api"
	"github.com/joefazee/neo/internal/security"
)

// AuthMiddleware now depends on the AuthService for permission fetching.
func AuthMiddleware(tokenMaker security.Maker, authService AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeaderKey)
		if authHeader == "" {
			api.UnauthorizedResponse(c)
			c.Abort()
			return
		}

		fields := strings.Fields(authHeader)
		if len(fields) < 2 || fields[0] != AuthorizationTypeBearer {
			api.UnauthorizedResponse(c)
			c.Abort()
			return
		}

		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			api.UnauthorizedResponse(c)
			c.Abort()
			return
		}

		permissions, err := authService.GetUserPermissions(c.Request.Context(), payload.UserID)
		if err != nil {
			api.ForbiddenResponse(c, "Could not retrieve user permissions")
			c.Abort()
			return
		}

		c.Set("userID", payload.UserID)
		c.Set("permissions", permissions)
		c.Next()
	}
}
