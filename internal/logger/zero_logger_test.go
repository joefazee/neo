package logger

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestLoggerInfo(t *testing.T) {
	var buffer bytes.Buffer
	logger := NewZeroLogger(&buffer, LevelInfo, nil)

	testCases := []struct {
		message    string
		properties Fields
	}{
		{"Test log message 1", Fields{"key1": "value1"}},
		{"Test log message 2", Fields{"key2": "value2"}},
		{"Test log message 3", Fields{"key3": "value3"}},
	}

	for _, tc := range testCases {
		logger.Info(tc.message, tc.properties)

		if !strings.Contains(buffer.String(), tc.message) {
			t.Errorf("Expected logger to have %v message in the buffer;", tc.message)
		}
	}
}

func TestLoggerError(t *testing.T) {
	var buffer bytes.Buffer
	logger := NewZeroLogger(&buffer, LevelError, nil)

	testCases := []struct {
		err        error
		properties Fields
	}{
		{errors.New("test error 1"), Fields{"key1": "value1"}},
		{errors.New("test error 2"), Fields{"key2": "value2"}},
		{errors.New("test error 3"), Fields{"key3": "value3"}},
	}

	for _, tc := range testCases {
		logger.Error(tc.err, tc.properties)

		if !strings.Contains(buffer.String(), tc.err.Error()) {
			t.Errorf("Expected logger to have %v message in the buffer;", tc.err.Error())
		}
	}
}

func TestLoggerSetLevel(t *testing.T) {
	var buffer bytes.Buffer
	logger := NewZeroLogger(&buffer, LevelInfo, nil)

	testCases := []struct {
		level Level
	}{
		{LevelError},
		{LevelFatal},
		{LevelOff},
	}

	for _, testCase := range testCases {
		logger.SetLevel(testCase.level)

		logger.Info("Test log message", Fields{"key": "value"})
		if buffer.String() != "" {
			t.Errorf("Expected logger not to write info logs with error level, but wrote: %s", buffer.String())
		}
	}
}
