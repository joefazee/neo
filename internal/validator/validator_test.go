package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	validator := New()

	require.NotNil(t, validator)
	require.NotNil(t, validator.Errors)
	require.Equal(t, 0, len(validator.Errors))
}

func TestValidator_AddError(t *testing.T) {
	validator := New()
	validator.AddError("name", "Name is required")
	if len(validator.Errors) != 1 {
		t.Error("validator.Errors should contain one entry")
	}
	if validator.Errors["name"] != "Name is required" {
		t.Error("validator.Errors[name] should contain the correct error message")
	}
}

func TestValidator_Check(t *testing.T) {
	validator := New()
	validator.Check(false, "name", "Name is required")
	if len(validator.Errors) != 1 {
		t.Error("validator.Errors should contain one entry")
	}
	if validator.Errors["name"] != "Name is required" {
		t.Error("validator.Errors[name] should contain the correct error message")
	}
}

func TestValidator_Valid(t *testing.T) {
	validator := New()
	if !validator.Valid() {
		t.Error("validator.Valid() should return true")
	}
	validator.Errors["name"] = "Name is required"
	if validator.Valid() {
		t.Error("validator.Valid() should return false")
	}
}
