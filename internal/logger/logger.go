package logger

type Fields map[string]interface{}

type Logger interface {
	Info(message string, properties map[string]interface{})
	Error(err error, properties map[string]interface{})
	Fatal(err error, properties map[string]interface{})
	Debug(message string, properties map[string]interface{})
	SetLevel(level Level)
}

type Level int8

const (
	LevelInfo Level = iota
	LevelError
	LevelFatal
	LevelOff
	LevelDebug
)

func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	case LevelDebug:
		return "DEBUG"
	default:
		return ""
	}
}
