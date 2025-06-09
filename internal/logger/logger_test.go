package logger

import (
	"testing"
)

func TestLevel_String(t *testing.T) {
	testCases := []struct {
		l        Level
		expected string
	}{
		{l: LevelInfo, expected: "INFO"},
		{l: LevelError, expected: "ERROR"},
		{l: LevelFatal, expected: "FATAL"},
		{l: LevelOff, expected: ""},
	}

	for _, tc := range testCases {
		got := tc.l.String()
		if got != tc.expected {
			t.Errorf("expected %v; got %v", tc.expected, got)
		}
	}
}
