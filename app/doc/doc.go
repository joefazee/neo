package doc

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/swag"
)

func serveSwaggerJSON(c *gin.Context) {
	// Get the original swagger JSON
	originalJSON, err := swag.ReadDoc()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read Swagger doc"})
		return
	}

	// Parse to modify it
	var swaggerData map[string]interface{}
	if err := json.Unmarshal([]byte(originalJSON), &swaggerData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Swagger doc"})
		return
	}

	// Add servers based on environment
	servers := getServersForEnvironment("development")
	swaggerData["servers"] = servers

	// Add security schemes
	if swaggerData["components"] == nil {
		swaggerData["components"] = make(map[string]interface{})
	}
	components := swaggerData["components"].(map[string]interface{})
	if components["securitySchemes"] == nil {
		components["securitySchemes"] = make(map[string]interface{})
	}
	securitySchemes := components["securitySchemes"].(map[string]interface{})
	securitySchemes["BearerAuth"] = map[string]interface{}{
		"type":         "http",
		"scheme":       "bearer",
		"bearerFormat": "JWT",
		"description":  "Enter JWT Bearer token",
	}

	// Marshal back to JSON
	modifiedJSON, err := json.Marshal(swaggerData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate modified Swagger doc"})
		return
	}

	c.Data(http.StatusOK, "application/json", modifiedJSON)
}

func getServersForEnvironment(environment string) []map[string]interface{} {
	servers := []map[string]interface{}{
		{
			"url":         "http://localhost:8080/api/v1",
			"description": "Local Development Server",
		},
	}

	if environment != "development" {
		servers = append(servers, map[string]interface{}{
			"url":         "https://staging.argue-and-earn.com/api/v1",
			"description": "Staging Server",
		})
	}

	if environment == "production" {
		servers = append(servers, map[string]interface{}{
			"url":         "https://argue-and-earn.com/api/v1",
			"description": "Production Server",
		})
	}

	return servers
}

func serveElements(c *gin.Context) {
	elementsHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Neo API Documentation</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <script src="https://unpkg.com/@stoplight/elements/web-components.min.js"></script>
    <link rel="stylesheet" href="https://unpkg.com/@stoplight/elements/styles.min.css">
    <style>
        body { margin: 0; padding: 0; height: 100vh; }
        elements-api { height: 100%; }
    </style>
</head>
<body>
    <elements-api
        apiDescriptionUrl="/swagger/doc.json"
        router="hash"
        layout="sidebar"
        tryItCredentialsPolicy="include"
        tryItCorsProxy=""
        hideInternal="false"
    ></elements-api>
</body>
</html>`
	c.Header("Content-Type", "text/html")
	c.String(200, elementsHTML)
}

func Init(r *gin.Engine) {
	r.GET("/swagger/doc.json", serveSwaggerJSON)

	r.GET("/docs/*any", serveElements)
}
