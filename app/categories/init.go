// app/categories/init.go
package categories

import (
	"github.com/gin-gonic/gin"
	"github.com/joefazee/neo/internal/deps"
)

const (
	CategoryRepoKey    = "category_repository"
	CategoryServiceKey = "category_service"
)

// MountPublic mounts public category routes
func MountPublic(r *gin.RouterGroup, container *deps.Container) {
	handler := createHandler(container)

	categoriesGroup := r.Group("/categories")
	categoriesGroup.GET("/:id", handler.GetCategoryByID)
	categoriesGroup.GET("/c/:countryId", handler.GetCategoriesByCountry)
	categoriesGroup.GET("/c/:countryId/:slug", handler.GetCategoryBySlug)
}

// MountAuthenticated mounts authenticated category routes
func MountAuthenticated(r *gin.RouterGroup, container *deps.Container) {
	handler := createHandler(container)

	categoriesGroup := r.Group("/categories")
	categoriesGroup.POST("", handler.CreateCategory)
	categoriesGroup.PUT("/:id", handler.UpdateCategory)
	categoriesGroup.DELETE("/:id", handler.DeleteCategory)
}

// InitRepositories initializes and registers repositories and services for this module
func InitRepositories(container *deps.Container) {
	// Initialize repository
	repo := NewRepository(container.DB)
	container.RegisterRepository(CategoryRepoKey, repo)

	// Initialize service
	service := NewService(repo)
	container.RegisterService(CategoryServiceKey, service)
}

// createHandler creates a category handler with all dependencies
func createHandler(container *deps.Container) *Handler {
	service := container.GetService(CategoryServiceKey).(Service)
	return NewHandler(service)
}
