package models

import (
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuditValues(t *testing.T) {
	t.Run("Value and Scan", func(t *testing.T) {
		values := AuditValues{
			"field1": "value1",
			"field2": 123,
			"field3": true,
		}

		value, err := values.Value()
		assert.NoError(t, err)

		var result AuditValues
		err = result.Scan(value)
		assert.NoError(t, err)
		assert.Equal(t, "value1", result["field1"])
		assert.Equal(t, float64(123), result["field2"])
		assert.Equal(t, true, result["field3"])

		jsonString := string(value.([]byte))
		err = result.Scan(jsonString)
		assert.NoError(t, err)

		err = result.Scan(nil)
		assert.NoError(t, err)

		err = result.Scan(42)
		assert.NoError(t, err)
	})

	t.Run("Value with nil", func(t *testing.T) {
		var values *AuditValues
		value, err := values.Value()
		assert.NoError(t, err)
		assert.Nil(t, value)
	})
}

func TestAuditLog(t *testing.T) {
	t.Run("TableName", func(t *testing.T) {
		al := AuditLog{}
		assert.Equal(t, "audit_logs", al.TableName())
	})

	t.Run("BeforeCreate", func(t *testing.T) {
		al := AuditLog{}
		assert.Equal(t, uuid.Nil, al.ID)

		err := al.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, al.ID)

		existingID := uuid.New()
		al2 := AuditLog{ID: existingID}
		err = al2.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, existingID, al2.ID)
	})

	t.Run("Action type checks", func(t *testing.T) {
		userID := uuid.New()

		userAction := AuditLog{UserID: &userID}
		assert.True(t, userAction.IsUserAction())
		assert.False(t, userAction.IsSystemAction())

		systemAction := AuditLog{UserID: nil}
		assert.False(t, systemAction.IsUserAction())
		assert.True(t, systemAction.IsSystemAction())
	})

	t.Run("GetChangedFields", func(t *testing.T) {
		oldValues := AuditValues{
			"name":   "John",
			"email":  "john@example.com",
			"status": "active",
		}
		newValues := AuditValues{
			"name":   "John Doe",
			"email":  "john@example.com",
			"status": "inactive",
			"phone":  "123456789",
		}

		al := AuditLog{
			OldValues: oldValues,
			NewValues: newValues,
		}

		changedFields := al.GetChangedFields()
		assert.Contains(t, changedFields, "name")
		assert.Contains(t, changedFields, "status")
		assert.Contains(t, changedFields, "phone")
		assert.NotContains(t, changedFields, "email")

		al.OldValues = nil
		changedFields = al.GetChangedFields()
		assert.Empty(t, changedFields)

		al.NewValues = nil
		changedFields = al.GetChangedFields()
		assert.Empty(t, changedFields)
	})

	t.Run("Validate", func(t *testing.T) {
		validAuditLog := AuditLog{
			Action:       "create",
			ResourceType: "user",
		}
		assert.NoError(t, validAuditLog.Validate())

		tests := []struct {
			name   string
			modify func(*AuditLog)
			err    error
		}{
			{"Valid AuditLog", func(_ *AuditLog) {}, nil},
			{"Empty Action", func(al *AuditLog) { al.Action = "" }, ErrInvalidAuditAction},
			{"Empty ResourceType", func(al *AuditLog) { al.ResourceType = "" }, ErrInvalidResourceType},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				auditLog := validAuditLog
				tt.modify(&auditLog)
				if tt.err != nil {
					assert.Equal(t, tt.err, auditLog.Validate())
				} else {
					assert.NoError(t, auditLog.Validate())
				}
			})
		}
	})

	t.Run("CreateUserAuditLog", func(t *testing.T) {
		userID := uuid.New()
		resourceID := uuid.New()
		oldValues := AuditValues{"status": "inactive"}
		newValues := AuditValues{"status": "active"}
		ip := net.ParseIP("192.168.1.1")

		al := CreateUserAuditLog(
			userID,
			"update",
			"user",
			&resourceID,
			oldValues,
			newValues,
			ip,
			"test-agent",
		)

		assert.Equal(t, userID, *al.UserID)
		assert.Equal(t, "update", al.Action)
		assert.Equal(t, "user", al.ResourceType)
		assert.Equal(t, resourceID, *al.ResourceID)
		assert.Equal(t, oldValues, al.OldValues)
		assert.Equal(t, newValues, al.NewValues)
		assert.Equal(t, ip, al.IPAddress)
		assert.Equal(t, "test-agent", al.UserAgent)
	})

	t.Run("CreateSystemAuditLog", func(t *testing.T) {
		resourceID := uuid.New()
		oldValues := AuditValues{"count": 10}
		newValues := AuditValues{"count": 15}

		al := CreateSystemAuditLog(
			"cleanup",
			"tokens",
			&resourceID,
			oldValues,
			newValues,
		)

		assert.Nil(t, al.UserID)
		assert.Equal(t, "cleanup", al.Action)
		assert.Equal(t, "tokens", al.ResourceType)
		assert.Equal(t, resourceID, *al.ResourceID)
		assert.Equal(t, oldValues, al.OldValues)
		assert.Equal(t, newValues, al.NewValues)
		assert.Equal(t, net.IP(nil), al.IPAddress)
		assert.Equal(t, "", al.UserAgent)
	})
}
