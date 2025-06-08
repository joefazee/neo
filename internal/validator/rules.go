package validator

import (
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	emailRegex = "^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$"
)

var (
	// EmailRgx is a regular expression for validating email addresses.
	EmailRgx = regexp.MustCompile(emailRegex)
)

// NotBlank returns true if a string is not empty or contains only whitespace.
func NotBlank(value string) bool {
	return strings.TrimSpace(value) != ""
}

// MinRunes returns true if a string is greater than or equal to a minimum number of n
func MinRunes(value string, n int) bool {
	return utf8.RuneCountInString(value) >= n
}

// MaxRunes returns true if a string is less than or equal to a maximum number of n
func MaxRunes(value string, n int) bool {
	return utf8.RuneCountInString(value) <= n
}

// Matches returns true if a string value matches a specific regexp pattern.
func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

// In returns true if a value is in a list of values.
func In[T comparable](value T, list ...T) bool {
	for i := range list {
		if value == list[i] {
			return true
		}
	}
	return false
}

// AllIn returns true if all values are in a list of values.
func AllIn[T comparable](values []T, safelist ...T) bool {
	for i := range values {
		if !In(values[i], safelist...) {
			return false
		}
	}
	return true
}

// NotIn returns true if a value is not in a list of values.
func NotIn[T comparable](value T, list ...T) bool {
	for i := range list {
		if value == list[i] {
			return false
		}
	}
	return true
}

// NoDuplicates returns true if all the values in a slice are unique.
func NoDuplicates[T comparable](values []T) bool {
	uniqueValues := make(map[T]bool)

	for _, value := range values {
		uniqueValues[value] = true
	}

	return len(values) == len(uniqueValues)
}

// IsEmail returns true if a string is a valid email address.
func IsEmail(value string) bool {
	if len(value) > 254 {
		return false
	}

	return EmailRgx.MatchString(value)
}

// IsURL returns true if a string is a valid URL.
func IsURL(value string) bool {
	u, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}

	return u.Scheme != "" && u.Host != ""
}

// Unique returns true if all the values in a slice are unique.
func Unique(values []string) bool {
	uniqueValues := make(map[string]bool)

	for _, value := range values {
		uniqueValues[value] = true
	}

	return len(values) == len(uniqueValues)
}

// IsValidTimeFormat returns true if a string is a valid time format.
func IsValidTimeFormat(timeStr string) bool {
	_, err := time.Parse("15:04", timeStr)
	return err == nil
}
