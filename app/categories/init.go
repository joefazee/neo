package categories

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Dependencies represent the dependencies needed for the categories module
type Dependencies struct {
	DB *gorm.DB
}

// Init initializes the categories module and mounts routes
func Init(r *gin.RouterGroup, deps Dependencies) {
	// Initialize repository
	repo := NewRepository(deps.DB)

	// Initialize service
	srvs := NewService(repo)

	// Initialize handler
	handler := NewHandler(srvs)

	// Mount standalone category routes
	categoriesGroup := r.Group("/categories")
	categoriesGroup.GET("/:id", handler.GetCategoryByID)
	categoriesGroup.GET("/c/:countryId", handler.GetCategoriesByCountry)
	categoriesGroup.GET("/c/:countryId/:slug", handler.GetCategoryBySlug)
}

// Init initializes the categories module and mounts routes
func InitWithAuth(r *gin.RouterGroup, deps Dependencies) {
	// Initialize repository
	repo := NewRepository(deps.DB)

	// Initialize service
	srvs := NewService(repo)

	// Initialize handler
	handler := NewHandler(srvs)

	// Mount standalone category routes
	categoriesGroup := r.Group("/categories")
	categoriesGroup.POST("", handler.CreateCategory)
	categoriesGroup.PUT("/:id", handler.UpdateCategory)
	categoriesGroup.DELETE("/:id", handler.DeleteCategory)
}
