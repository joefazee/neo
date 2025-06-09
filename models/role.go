package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role represents a user role in the system
type Role struct {
	ID          uuid.UUID    `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name        string       `gorm:"type:varchar(50);not null;unique"`
	Description string       `gorm:"type:text"`
	Permissions []Permission `gorm:"many2many:role_permissions;"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// BeforeCreate sets a UUID for the role before creation.
func (r *Role) BeforeCreate(_ *gorm.DB) (err error) {
	r.ID = uuid.New()
	return
}

// Permission represents an action that can be performed
type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name        string    `gorm:"type:varchar(50);not null;unique"`
	Description string    `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// BeforeCreate sets a UUID for the permission before creation.
func (p *Permission) BeforeCreate(_ *gorm.DB) (err error) {
	p.ID = uuid.New()
	return
}
