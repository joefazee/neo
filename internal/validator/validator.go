package validator

import (
	"context"
)

type ErrorFields map[string]string

// ValidationError Custom error type for better error handling
type ValidationError struct {
	Fields  map[string]string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type Validator struct {
	Errors ErrorFields
	ctx    context.Context
}

// New returns a new Validator instance.
func New() *Validator {
	return &Validator{Errors: make(ErrorFields)}
}

func NewValidationError(message string, fields ErrorFields) *ValidationError {
	return &ValidationError{
		Fields:  fields,
		Message: message,
	}
}

// NewWithStore returns a new Validator instance with a store.
func NewWithStore(ctx context.Context) *Validator {
	return &Validator{
		Errors: make(ErrorFields),
		ctx:    ctx,
	}
}

// Valid returns true if the errors map doesn't contain any entries.
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// AddError adds an error message to the map (so long as no entry already exists for
// the given key).
func (v *Validator) AddError(key, message string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

// Check adds an error message to the map only if a validation check is not 'ok'.
func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}
}
