package logger

import (
	"io"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ZeroLogger struct {
	writer        io.Writer
	level         Level
	defaultFields Fields
}

type CallerHook struct{}

func (h CallerHook) Run(e *zerolog.Event, level zerolog.Level, _ string) {
	if _, file, line, ok := runtime.Caller(5); ok {
		e.Str("file", file)
		e.Int("line", line)
		e.Int("level", int(level))
	}
}

// NewZeroLogger return a configured instance of NewZeroLogger
func NewZeroLogger(writer io.Writer, level Level, defaultFields Fields) *ZeroLogger {
	if defaultFields == nil {
		defaultFields = Fields{}
	}
	zeroLogger := ZeroLogger{writer: writer, level: level, defaultFields: defaultFields}
	zeroLogger.configureLogger()
	return &zeroLogger
}

func (l *ZeroLogger) configureLogger() {
	var zLevel zerolog.Level
	switch l.level {
	case LevelInfo:
		zLevel = zerolog.InfoLevel
	case LevelError:
		zLevel = zerolog.ErrorLevel
	case LevelFatal:
		zLevel = zerolog.FatalLevel
	case LevelOff:
		zLevel = zerolog.Disabled
	default:
		zLevel = zerolog.InfoLevel
	}

	props := make(map[string]interface{})
	for k, v := range l.defaultFields {
		props[k] = v
	}

	log.Logger = zerolog.New(l.writer).With().Fields(props).Timestamp().Logger().Level(zLevel)
}

// Info only logs information
func (l *ZeroLogger) Info(message string, properties map[string]interface{}) {
	log.Info().Fields(properties).Msg(message)
}

// Error reports all error at error level
func (l *ZeroLogger) Error(err error, properties map[string]interface{}) {
	log.Error().Fields(properties).Err(err).Msg(err.Error())
}

// Fatal write the log to output and stop the process
func (l *ZeroLogger) Fatal(err error, properties map[string]interface{}) {
	log.Fatal().Fields(properties).Err(err).Msg(err.Error())
}

// Debug this is for debugging and we use it to store some information in the log
func (l *ZeroLogger) Debug(message string, properties map[string]interface{}) {
	log.Debug().Fields(properties).Msg(message)
}

func (l *ZeroLogger) SetLevel(level Level) {
	l.level = level
	l.configureLogger()
}
