package user

import (
	"errors"
)

type Config struct {
	SymmetricKey string `env:"SYMMETRIC_KEY"`
}

func (c *Config) Validate() error {
	if c.SymmetricKey == "" {
		return errors.New("symmetric key must be set")
	}
	return nil
}

func GetDefaultConfig() *Config {
	return &Config{
		SymmetricKey: "12345678901234567890123456789012",
	}
}
