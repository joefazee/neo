package models

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestTokenBlacklist(t *testing.T) {
	t.Run("TableName", func(t *testing.T) {
		tb := TokenBlacklist{}
		assert.Equal(t, "token_blacklist", tb.TableName())
	})

	t.Run("BeforeCreate", func(t *testing.T) {
		tb := TokenBlacklist{}
		assert.Equal(t, uuid.Nil, tb.ID)

		err := tb.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, tb.ID)

		existingID := uuid.New()
		tb2 := TokenBlacklist{ID: existingID}
		err = tb2.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, existingID, tb2.ID)
	})

	t.Run("IsExpired", func(t *testing.T) {
		futureTime := time.Now().Add(time.Hour)
		pastTime := time.Now().Add(-time.Hour)

		tb := TokenBlacklist{ExpiresAt: futureTime}
		assert.False(t, tb.IsExpired())

		tb.ExpiresAt = pastTime
		assert.True(t, tb.IsExpired())
	})

	t.Run("ShouldCleanup", func(t *testing.T) {
		recentExpiry := time.Now().Add(-time.Hour)
		oldExpiry := time.Now().Add(-25 * time.Hour)

		tb := TokenBlacklist{ExpiresAt: recentExpiry}
		assert.False(t, tb.ShouldCleanup())

		tb.ExpiresAt = oldExpiry
		assert.True(t, tb.ShouldCleanup())
	})

	t.Run("Validate", func(t *testing.T) {
		validToken := TokenBlacklist{
			TokenJTI:  "jti_123",
			UserID:    uuid.New(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		assert.NoError(t, validToken.Validate())

		tests := []struct {
			name   string
			modify func(*TokenBlacklist)
			err    error
		}{
			{"Empty TokenJTI", func(tb *TokenBlacklist) { tb.TokenJTI = "" }, ErrInvalidTokenJTI},
			{"Invalid UserID", func(tb *TokenBlacklist) { tb.UserID = uuid.Nil }, ErrInvalidUserID},
			{"Expired Token", func(tb *TokenBlacklist) { tb.ExpiresAt = time.Now().Add(-time.Hour) }, ErrTokenAlreadyExpired},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				token := validToken
				tt.modify(&token)
				assert.Equal(t, tt.err, token.Validate())
			})
		}
	})

	t.Run("CreateBlacklistEntry", func(t *testing.T) {
		jti := "jti_12345"
		userID := uuid.New()
		expiresAt := time.Now().Add(time.Hour)

		entry := CreateBlacklistEntry(jti, userID, expiresAt)

		assert.Equal(t, jti, entry.TokenJTI)
		assert.Equal(t, userID, entry.UserID)
		assert.Equal(t, expiresAt, entry.ExpiresAt)
	})

	t.Run("CleanupExpiredTokens", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: db,
		}), &gorm.Config{})
		assert.NoError(t, err)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "token_blacklist" WHERE expires_at < \$1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 3))
		mock.ExpectCommit()

		err = CleanupExpiredTokens(gormDB)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
