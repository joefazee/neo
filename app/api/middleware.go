package api

import "github.com/gin-gonic/gin"

func Can(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		permissionsValue, exists := c.Get("permissions")
		if !exists {
			ForbiddenResponse(c, "Access Denied: Permissions not found in context")
			c.Abort()
			return
		}

		permissions, ok := permissionsValue.([]string)
		if !ok {
			ForbiddenResponse(c, "Access Denied: Invalid permissions data in context")
			c.Abort()
			return
		}

		for _, p := range permissions {
			if p == permission {
				c.Next()
				return
			}
		}

		ForbiddenResponse(c, "Access Denied: You do not have the required permission")
		c.Abort()
	}
}
