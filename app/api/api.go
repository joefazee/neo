package api

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

var allowedHeaders = "Content-Type, " +
	"Content-Length, " +
	"Accept-Encoding, " +
	"X-CSRF-Token, " +
	"Authorization, " +
	"accept, origin, " +
	"Cache-Control, " +
	"X-Requested-With"

func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// HealthCheck returns the health status of the API
// @Summary Health Check
// @Description Check if the API is running
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/healthz [get]
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":      "healthy",
		"environment": os.Getenv("APP_ENV"),
		"version":     "1.0.0",
	})
}
