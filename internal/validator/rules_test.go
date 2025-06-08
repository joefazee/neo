package validator

import "testing"

func TestNotBlank(t *testing.T) {
	validator := New()
	validator.Check(NotBlank(""), "name", "Name is required")
	if validator.Valid() {
		t.Error("validator.Valid() should return false")
	}
	if len(validator.Errors) != 1 {
		t.Error("validator.Errors should contain one entry")
	}
	if validator.Errors["name"] != "Name is required" {
		t.Error("validator.Errors[name] should contain the correct error message")
	}
}
