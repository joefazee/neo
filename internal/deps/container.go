package deps

import (
	"github.com/joefazee/neo/internal/cache"
	"github.com/joefazee/neo/internal/logger"
	"github.com/joefazee/neo/internal/sanitizer"
	"github.com/joefazee/neo/internal/security"
	"gorm.io/gorm"
)

// Container holds all shared dependencies
type Container struct {
	DB         *gorm.DB
	TokenMaker security.Maker
	Sanitizer  sanitizer.HTMLStripperer
	Logger     logger.Logger
	Cache      cache.Cache[string]

	// Store repositories as interfaces to avoid imports
	repositories map[string]interface{}
	services     map[string]interface{}
}

func NewContainer(db *gorm.DB, tokenMaker security.Maker, sanitizer sanitizer.HTMLStripperer, logger logger.Logger, cache cache.Cache[string]) *Container {
	return &Container{
		DB:           db,
		TokenMaker:   tokenMaker,
		Sanitizer:    sanitizer,
		Logger:       logger,
		Cache:        cache,
		repositories: make(map[string]interface{}),
		services:     make(map[string]interface{}),
	}
}

// RegisterRepository stores a repository with a key
func (c *Container) RegisterRepository(key string, repo interface{}) {
	c.repositories[key] = repo
}

// GetRepository retrieves a repository by key
func (c *Container) GetRepository(key string) interface{} {
	return c.repositories[key]
}

// RegisterService stores a service with a key
func (c *Container) RegisterService(key string, service interface{}) {
	c.services[key] = service
}

// GetService retrieves a service by key
func (c *Container) GetService(key string) interface{} {
	return c.services[key]
}
