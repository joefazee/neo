package models

import (
	"database/sql/driver"
	"encoding/json"
	"net"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditValues represents values for audit logging
type AuditValues map[string]interface{}

// AuditLog represents an audit trail entry for security and compliance
type AuditLog struct {
	ID           uuid.UUID   `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID       *uuid.UUID  `gorm:"type:uuid;index:idx_audit_logs_user" json:"user_id"`
	Action       string      `gorm:"type:varchar(50);not null" json:"action"`
	ResourceType string      `gorm:"type:varchar(50);not null" json:"resource_type"`
	ResourceID   *uuid.UUID  `gorm:"type:uuid" json:"resource_id"`
	OldValues    AuditValues `gorm:"type:jsonb" json:"old_values"`
	NewValues    AuditValues `gorm:"type:jsonb" json:"new_values"`
	IPAddress    net.IP      `gorm:"type:inet" json:"ip_address"`
	UserAgent    string      `gorm:"type:text" json:"user_agent"`
	CreatedAt    time.Time   `gorm:"autoCreateTime;index:idx_audit_logs_created_at" json:"created_at"`

	// Associations
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName specifies the table name for AuditLog model
func (*AuditLog) TableName() string {
	return "audit_logs"
}

// BeforeCreate sets up the model before creation
func (al *AuditLog) BeforeCreate(_ *gorm.DB) error {
	if al.ID == uuid.Nil {
		al.ID = uuid.New()
	}
	return nil
}

// Value implements driver.Valuer interface for AuditValues
func (av *AuditValues) Value() (driver.Value, error) {
	if av == nil {
		return nil, nil
	}
	return json.Marshal(av)
}

// Scan implements sql.Scanner interface for AuditValues
func (av *AuditValues) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, av)
	case string:
		return json.Unmarshal([]byte(v), av)
	}
	return nil
}

// IsUserAction checks if this audit log is associated with a user
func (al *AuditLog) IsUserAction() bool {
	return al.UserID != nil
}

// IsSystemAction checks if this audit log is a system action
func (al *AuditLog) IsSystemAction() bool {
	return al.UserID == nil
}

// GetChangedFields returns the fields that were changed
func (al *AuditLog) GetChangedFields() []string {
	if al.OldValues == nil || al.NewValues == nil {
		return []string{}
	}

	var changedFields []string
	for field := range al.NewValues {
		if oldVal, exists := al.OldValues[field]; !exists || oldVal != al.NewValues[field] {
			changedFields = append(changedFields, field)
		}
	}

	return changedFields
}

// Validate performs validation on the audit log model
func (al *AuditLog) Validate() error {
	if al.Action == "" {
		return ErrInvalidAuditAction
	}
	if al.ResourceType == "" {
		return ErrInvalidResourceType
	}
	return nil
}

// CreateUserAuditLog creates an audit log entry for user actions
func CreateUserAuditLog(userID uuid.UUID,
	action,
	resourceType string,
	resourceID *uuid.UUID,
	oldValues, newValues AuditValues,
	ip net.IP, userAgent string) *AuditLog {
	return &AuditLog{
		UserID:       &userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		OldValues:    oldValues,
		NewValues:    newValues,
		IPAddress:    ip,
		UserAgent:    userAgent,
	}
}

// CreateSystemAuditLog creates an audit log entry for system actions
func CreateSystemAuditLog(action, resourceType string,
	resourceID *uuid.UUID,
	oldValues, newValues AuditValues) *AuditLog {
	return &AuditLog{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		OldValues:    oldValues,
		NewValues:    newValues,
	}
}
