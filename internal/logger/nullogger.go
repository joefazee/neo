package logger

// NullLogger is a no-op implementation of the Logger interface.
type NullLogger struct{}

// Ensure NullLogger implements Logger.
var _ Logger = (*NullLogger)(nil)

// NewNullLogger returns an instance of NullLogger.
func NewNullLogger() *NullLogger {
	return &NullLogger{}
}

// Info does nothing.
func (l *NullLogger) Info(_ string, _ map[string]interface{}) {}

// Error does nothing.
func (l *NullLogger) Error(_ error, _ map[string]interface{}) {}

// Fatal does nothing.
func (l *NullLogger) Fatal(_ error, _ map[string]interface{}) {}

// Debug does nothing.
func (l *NullLogger) Debug(_ string, _ map[string]interface{}) {}

// SetLevel does nothing.
func (l *NullLogger) SetLevel(_ Level) {}
