package countries

import (
	"github.com/gin-gonic/gin"
	"github.com/joefazee/neo/internal/deps"
)

const (
	CountryRepoKey = "country_repository"
)

// MountPublic mounts public country routes
func MountPublic(r *gin.RouterGroup, container *deps.Container) {
	handler := createHandler(container)

	countriesGroup := r.Group("/countries")
	countriesGroup.GET("", handler.GetAllCountries)
	countriesGroup.GET("/active", handler.GetActiveCountries)
	countriesGroup.GET("/:id", handler.GetCountryByID)
	countriesGroup.GET("/code/:code", handler.GetCountryByCode)
}

// MountAuthenticated mounts authenticated country routes
func MountAuthenticated(r *gin.RouterGroup, container *deps.Container) {
	handler := createHandler(container)

	countriesGroup := r.Group("/countries")
	countriesGroup.POST("", handler.CreateCountry)
	countriesGroup.PUT("/:id", handler.UpdateCountry)
	countriesGroup.DELETE("/:id", handler.DeleteCountry)
}

// InitRepositories initializes and registers repositories for this module
func InitRepositories(container *deps.Container) {
	repo := NewRepository(container.DB)
	container.RegisterRepository(CountryRepoKey, repo)
}

// createHandler creates a handler with all dependencies
func createHandler(container *deps.Container) *Handler {
	// Get repository from container
	repo := container.GetRepository(CountryRepoKey).(Repository)

	// Create service
	service := NewService(repo)

	// Create handler
	return NewHandler(service)
}
