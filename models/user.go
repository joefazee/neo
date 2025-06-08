package models

import (
	"database/sql/driver"
	"encoding/json"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// KYCStatus represents the KYC verification status
type KYCStatus string

const (
	KYCStatusPending    KYCStatus = "pending"
	KYCStatusInProgress KYCStatus = "in_progress"
	KYCStatusVerified   KYCStatus = "verified"
	KYCStatusRejected   KYCStatus = "rejected"
)

// UserMetadata represents additional user metadata
type UserMetadata struct {
	ReferralCode   string    `json:"referral_code,omitempty"`
	ReferredBy     string    `json:"referred_by,omitempty"`
	PreferredLang  string    `json:"preferred_lang,omitempty"`
	NewsletterSubs bool      `json:"newsletter_subscribed,omitempty"`
	LastSeenAt     time.Time `json:"last_seen_at,omitempty"`
}

// Value implements driver.Valuer interface
func (m *UserMetadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements sql.Scanner interface
func (m *UserMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, m)
	case string:
		return json.Unmarshal([]byte(v), m)
	}
	return nil
}

// User represents a user in the system
type User struct {
	ID                  uuid.UUID    `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	CountryID           uuid.UUID    `gorm:"type:uuid;not null;index" json:"country_id"`
	Email               string       `gorm:"type:varchar(255);not null;unique;index" json:"email"`
	EmailVerifiedAt     *time.Time   `gorm:"type:timestamptz" json:"email_verified_at"`
	PasswordHash        string       `gorm:"type:varchar(255);not null" json:"-"` // Never expose password
	FirstName           string       `gorm:"type:varchar(100)" json:"first_name"`
	LastName            string       `gorm:"type:varchar(100)" json:"last_name"`
	Phone               string       `gorm:"type:varchar(20)" json:"phone"`
	PhoneVerifiedAt     *time.Time   `gorm:"type:timestamptz" json:"phone_verified_at"`
	DateOfBirth         *time.Time   `gorm:"type:date" json:"date_of_birth"`
	KYCStatus           KYCStatus    `gorm:"type:varchar(20);default:'pending';index" json:"kyc_status"`
	KYCProvider         string       `gorm:"type:varchar(50)" json:"kyc_provider"`
	KYCReference        string       `gorm:"type:varchar(100)" json:"kyc_reference"`
	KYCVerifiedAt       *time.Time   `gorm:"type:timestamptz" json:"kyc_verified_at"`
	TwoFactorEnabled    bool         `gorm:"default:false" json:"two_factor_enabled"`
	TwoFactorSecret     string       `gorm:"type:varchar(255)" json:"-"` // Never expose 2FA secret
	LastLoginAt         *time.Time   `gorm:"type:timestamptz" json:"last_login_at"`
	LastLoginIP         net.IP       `gorm:"type:inet" json:"last_login_ip"`
	FailedLoginAttempts int          `gorm:"default:0" json:"failed_login_attempts"`
	LockedUntil         *time.Time   `gorm:"type:timestamptz" json:"locked_until"`
	IsActive            *bool        `gorm:"default:true" json:"is_active"`
	Metadata            UserMetadata `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt           time.Time    `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time    `gorm:"autoUpdateTime" json:"updated_at"`

	// Associations
	Country             *Country             `gorm:"foreignKey:CountryID" json:"country,omitempty"`
	Wallets             []Wallet             `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"wallets,omitempty"`
	Bets                []Bet                `gorm:"foreignKey:UserID" json:"-"`
	CreatedMarkets      []Market             `gorm:"foreignKey:CreatorID" json:"-"`
	Transactions        []Transaction        `gorm:"foreignKey:UserID" json:"-"`
	PaymentTransactions []PaymentTransaction `gorm:"foreignKey:UserID" json:"-"`
	Settlements         []Settlement         `gorm:"foreignKey:UserID" json:"-"`
}

// TableName specifies the table name for User model
func (*User) TableName() string {
	return "users"
}

// BeforeCreate sets up the model before creation
func (u *User) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// SetPassword hashes and sets the user password
func (u *User) SetPassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword verifies the provided password against the stored hash
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// IsEmailVerified checks if the user's email is verified
func (u *User) IsEmailVerified() bool {
	return u.EmailVerifiedAt != nil
}

// IsPhoneVerified checks if the user's phone is verified
func (u *User) IsPhoneVerified() bool {
	return u.PhoneVerifiedAt != nil
}

// IsKYCVerified checks if the user's KYC is verified
func (u *User) IsKYCVerified() bool {
	return u.KYCStatus == KYCStatusVerified && u.KYCVerifiedAt != nil
}

// IsLocked checks if the user account is currently locked
func (u *User) IsLocked() bool {
	return u.LockedUntil != nil && u.LockedUntil.After(time.Now())
}

// CanWithdraw checks if user is allowed to withdraw funds
func (u *User) CanWithdraw() bool {
	return u.IsEmailVerified() && u.IsKYCVerified() && *u.IsActive && !u.IsLocked()
}

// CanBet checks if user is allowed to place bets
func (u *User) CanBet() bool {
	return u.IsEmailVerified() && *u.IsActive && !u.IsLocked()
}

// IncrementFailedLogins increments the failed login counter
func (u *User) IncrementFailedLogins() {
	u.FailedLoginAttempts++
	if u.FailedLoginAttempts >= 5 {
		lockDuration := time.Hour // Lock for 1 hour after 5 failed attempts
		lockUntil := time.Now().Add(lockDuration)
		u.LockedUntil = &lockUntil
	}
}

// ResetFailedLogins resets the failed login counter
func (u *User) ResetFailedLogins() {
	u.FailedLoginAttempts = 0
	u.LockedUntil = nil
}

// UpdateLastLogin updates the last login information
func (u *User) UpdateLastLogin(ip net.IP) {
	now := time.Now()
	u.LastLoginAt = &now
	u.LastLoginIP = ip
	u.ResetFailedLogins()
}

// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	if u.FirstName == "" && u.LastName == "" {
		return ""
	}
	return u.FirstName + " " + u.LastName
}

// Validate performs validation on the user model
func (u *User) Validate() error {
	if u.CountryID == uuid.Nil {
		return ErrInvalidCountryID
	}
	if u.Email == "" {
		return ErrInvalidEmail
	}
	if u.PasswordHash == "" {
		return ErrInvalidPassword
	}
	return nil
}

// MaskSensitiveData masks sensitive information for logging/auditing
func (u *User) MaskSensitiveData() *User {
	masked := *u
	masked.PasswordHash = "***"
	masked.TwoFactorSecret = "***"
	masked.Email = "***" + masked.Email[len(masked.Email)-4:]
	if len(masked.Phone) > 4 {
		masked.Phone = "***" + masked.Phone[len(masked.Phone)-4:]
	}
	return &masked
}

func IsEmail(identity string) bool {
	return identity != "" && strings.Contains(identity, "@") && strings.Contains(identity, ".")
}

func CheckPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", ErrPasswordTooShort
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
