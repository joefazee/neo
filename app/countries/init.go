package countries

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Dependencies represents the dependencies needed for the countries module
type Dependencies struct {
	DB *gorm.DB
}

// Init initializes the countries module and mounts routes
func Init(r *gin.RouterGroup, deps Dependencies) {
	// Initialize repository
	repo := NewRepository(deps.DB)

	// Initialize service
	srvs := NewService(repo)

	// Initialize handler
	handler := NewHandler(srvs)

	// Mount routes
	countriesGroup := r.Group("/countries")
	countriesGroup.GET("", handler.GetAllCountries)
	countriesGroup.GET("/active", handler.GetActiveCountries)
	countriesGroup.GET("/:id", handler.GetCountryByID)
	countriesGroup.GET("/code/:code", handler.GetCountryByCode)
	countriesGroup.POST("", handler.CreateCountry)
	countriesGroup.PUT("/:id", handler.UpdateCountry)
	countriesGroup.DELETE("/:id", handler.DeleteCountry)
}
