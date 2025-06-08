package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Category represents a market category within a specific country
type Category struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	CountryID   uuid.UUID `gorm:"type:uuid;not null;index:idx_categories_country_slug" json:"country_id"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Slug        string    `gorm:"type:varchar(100);not null;index:idx_categories_country_slug" json:"slug"`
	Description string    `gorm:"type:text" json:"description"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	SortOrder   int       `gorm:"default:0" json:"sort_order"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Associations
	Country *Country `gorm:"foreignKey:CountryID;constraint:OnDelete:CASCADE" json:"country,omitempty"`
	Markets []Market `gorm:"foreignKey:CategoryID" json:"-"`
}

// TableName specifies the table name for Category model
func (*Category) TableName() string {
	return "categories"
}

// BeforeCreate sets up the model before creation
func (c *Category) BeforeCreate(_ *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// Validate performs validation on the category model
func (c *Category) Validate() error {
	if c.CountryID == uuid.Nil {
		return ErrInvalidCountryID
	}
	if c.Name == "" {
		return ErrInvalidCategoryName
	}
	if c.Slug == "" {
		return ErrInvalidCategorySlug
	}
	return nil
}

// IsValidSlug checks if the slug contains only valid characters
func (c *Category) IsValidSlug() bool {
	// Basic slug validation - alphanumeric and hyphens only
	for _, char := range c.Slug {
		if !((char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '-') {
			return false
		}
	}
	return c.Slug != ""
}

// GetActiveMarkets returns active markets for this category
func (c *Category) GetActiveMarkets(db *gorm.DB) ([]Market, error) {
	var markets []Market
	err := db.Where("category_id = ? AND status IN ?", c.ID, []string{"open", "closed"}).Find(&markets).Error
	return markets, err
}

// GetMarketCount returns the total number of markets in this category
func (c *Category) GetMarketCount(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Model(&Market{}).Where("category_id = ?", c.ID).Count(&count).Error
	return count, err
}
