package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	config := GetDefaultConfig()

	err := config.Validate()
	assert.NoError(t, err, "Expected no error for valid config")
	config.SymmetricKey = ""
	err = config.Validate()
	assert.Error(t, err, "Expected error for invalid config")
}
