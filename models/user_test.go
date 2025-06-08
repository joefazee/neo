package models

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUserMetadata(t *testing.T) {
	t.Run("Value and Scan", func(t *testing.T) {
		lastSeen := time.Now()
		metadata := UserMetadata{
			ReferralCode:   "REF123",
			ReferredBy:     "user456",
			PreferredLang:  "en",
			NewsletterSubs: true,
			LastSeenAt:     lastSeen,
		}

		value, err := metadata.Value()
		assert.NoError(t, err)

		var result UserMetadata
		err = result.Scan(value)
		assert.NoError(t, err)
		assert.Equal(t, metadata.ReferralCode, result.ReferralCode)
		assert.Equal(t, metadata.PreferredLang, result.PreferredLang)
		assert.Equal(t, metadata.NewsletterSubs, result.NewsletterSubs)

		jsonString := string(value.([]byte))
		err = result.Scan(jsonString)
		assert.NoError(t, err)

		err = result.Scan(nil)
		assert.NoError(t, err)

		err = result.Scan(42)
		assert.NoError(t, err)
	})
}

func TestUser(t *testing.T) {
	t.Run("TableName", func(t *testing.T) {
		u := User{}
		assert.Equal(t, "users", u.TableName())
	})

	t.Run("BeforeCreate", func(t *testing.T) {
		u := User{}
		assert.Equal(t, uuid.Nil, u.ID)

		err := u.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, u.ID)

		existingID := uuid.New()
		u2 := User{ID: existingID}
		err = u2.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, existingID, u2.ID)
	})

	t.Run("Password management", func(t *testing.T) {
		u := User{}

		err := u.SetPassword("short")
		assert.Equal(t, ErrPasswordTooShort, err)

		err = u.SetPassword("validpassword123")
		assert.NoError(t, err)
		assert.NotEmpty(t, u.PasswordHash)

		assert.True(t, u.CheckPassword("validpassword123"))
		assert.False(t, u.CheckPassword("wrongpassword"))

		longPassword := strings.Repeat("a", 73)
		err = u.SetPassword(longPassword)
		assert.Error(t, err)
	})

	t.Run("Verification status checks", func(t *testing.T) {
		u := User{}

		assert.False(t, u.IsEmailVerified())
		now := time.Now()
		u.EmailVerifiedAt = &now
		assert.True(t, u.IsEmailVerified())

		assert.False(t, u.IsPhoneVerified())
		u.PhoneVerifiedAt = &now
		assert.True(t, u.IsPhoneVerified())

		assert.False(t, u.IsKYCVerified())
		u.KYCStatus = KYCStatusVerified
		u.KYCVerifiedAt = &now
		assert.True(t, u.IsKYCVerified())
	})

	t.Run("Account lock status", func(t *testing.T) {
		u := User{}

		assert.False(t, u.IsLocked())

		future := time.Now().Add(time.Hour)
		u.LockedUntil = &future
		assert.True(t, u.IsLocked())

		past := time.Now().Add(-time.Hour)
		u.LockedUntil = &past
		assert.False(t, u.IsLocked())
	})

	t.Run("Permission checks", func(t *testing.T) {
		isActive := true
		now := time.Now()
		u := User{
			EmailVerifiedAt: &now,
			KYCStatus:       KYCStatusVerified,
			KYCVerifiedAt:   &now,
			IsActive:        &isActive,
		}

		assert.True(t, u.CanWithdraw())
		assert.True(t, u.CanBet())

		isActive = false
		u.IsActive = &isActive
		assert.False(t, u.CanWithdraw())
		assert.False(t, u.CanBet())

		isActive = true
		u.IsActive = &isActive
		u.EmailVerifiedAt = nil
		assert.False(t, u.CanWithdraw())
		assert.False(t, u.CanBet())
	})

	t.Run("Failed login management", func(t *testing.T) {
		u := User{}

		for i := 0; i < 4; i++ {
			u.IncrementFailedLogins()
			assert.Equal(t, i+1, u.FailedLoginAttempts)
			assert.Nil(t, u.LockedUntil)
		}

		u.IncrementFailedLogins()
		assert.Equal(t, 5, u.FailedLoginAttempts)
		assert.NotNil(t, u.LockedUntil)
		assert.True(t, u.LockedUntil.After(time.Now()))

		u.ResetFailedLogins()
		assert.Equal(t, 0, u.FailedLoginAttempts)
		assert.Nil(t, u.LockedUntil)
	})

	t.Run("UpdateLastLogin", func(t *testing.T) {
		u := User{FailedLoginAttempts: 3}
		ip := net.ParseIP("192.168.1.1")

		u.UpdateLastLogin(ip)

		assert.NotNil(t, u.LastLoginAt)
		assert.True(t, u.LastLoginAt.After(time.Now().Add(-time.Minute)))
		assert.Equal(t, ip, u.LastLoginIP)
		assert.Equal(t, 0, u.FailedLoginAttempts)
	})

	t.Run("GetFullName", func(t *testing.T) {
		tests := []struct {
			firstName string
			lastName  string
			expected  string
		}{
			{"John", "Doe", "John Doe"},
			{"John", "", "John "},
			{"", "Doe", " Doe"},
			{"", "", ""},
		}

		for _, tt := range tests {
			u := User{FirstName: tt.firstName, LastName: tt.lastName}
			assert.Equal(t, tt.expected, u.GetFullName())
		}
	})

	t.Run("Validate", func(t *testing.T) {
		validUser := User{
			CountryID:    uuid.New(),
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
		}
		assert.NoError(t, validUser.Validate())

		tests := []struct {
			name   string
			modify func(*User)
			err    error
		}{
			{"Valid User", func(_ *User) {}, nil},
			{"Invalid CountryID", func(u *User) { u.CountryID = uuid.Nil }, ErrInvalidCountryID},
			{"Empty Email", func(u *User) { u.Email = "" }, ErrInvalidEmail},
			{"Empty Password", func(u *User) { u.PasswordHash = "" }, ErrInvalidPassword},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				user := validUser
				tt.modify(&user)
				if tt.err != nil {
					assert.Equal(t, tt.err, user.Validate())
				} else {
					assert.NoError(t, user.Validate())
				}
			})
		}
	})

	t.Run("MaskSensitiveData", func(t *testing.T) {
		u := User{
			Email:           "test@example.com",
			Phone:           "1234567890",
			PasswordHash:    "secret_hash",
			TwoFactorSecret: "secret_2fa",
		}

		masked := u.MaskSensitiveData()

		assert.Equal(t, "***", masked.PasswordHash)
		assert.Equal(t, "***", masked.TwoFactorSecret)
		assert.Equal(t, "***.com", masked.Email)
		assert.Equal(t, "***7890", masked.Phone)

		assert.NotEqual(t, &u, masked)
	})
}
