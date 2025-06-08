package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TokenBlacklist represents blacklisted JWT tokens for security
type TokenBlacklist struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	TokenJTI  string    `gorm:"type:varchar(255);not null;unique;index:idx_token_blacklist_jti" json:"token_jti"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	ExpiresAt time.Time `gorm:"type:timestamptz;not null;index:idx_token_blacklist_expires_at" json:"expires_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// Associations
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName specifies the table name for TokenBlacklist model
func (*TokenBlacklist) TableName() string {
	return "token_blacklist"
}

// BeforeCreate sets up the model before creation
func (tb *TokenBlacklist) BeforeCreate(_ *gorm.DB) error {
	if tb.ID == uuid.Nil {
		tb.ID = uuid.New()
	}
	return nil
}

// IsExpired checks if the blacklisted token has expired
func (tb *TokenBlacklist) IsExpired() bool {
	return time.Now().After(tb.ExpiresAt)
}

// ShouldCleanup checks if this blacklist entry can be cleaned up
func (tb *TokenBlacklist) ShouldCleanup() bool {
	// Clean up expired tokens after a grace period
	gracePeriod := 24 * time.Hour
	return time.Now().After(tb.ExpiresAt.Add(gracePeriod))
}

// Validate performs validation on the token blacklist model
func (tb *TokenBlacklist) Validate() error {
	if tb.TokenJTI == "" {
		return ErrInvalidTokenJTI
	}
	if tb.UserID == uuid.Nil {
		return ErrInvalidUserID
	}
	if tb.ExpiresAt.Before(time.Now()) {
		return ErrTokenAlreadyExpired
	}
	return nil
}

// CreateBlacklistEntry creates a new blacklist entry
func CreateBlacklistEntry(jti string, userID uuid.UUID, expiresAt time.Time) *TokenBlacklist {
	return &TokenBlacklist{
		TokenJTI:  jti,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
}

// CleanupExpiredTokens removes expired tokens from blacklist (use in background job)
func CleanupExpiredTokens(db *gorm.DB) error {
	cutoff := time.Now().Add(-24 * time.Hour)
	return db.Where("expires_at < ?", cutoff).Delete(&TokenBlacklist{}).Error
}
