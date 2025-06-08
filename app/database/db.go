package database

import (
	"fmt"

	"github.com/joefazee/neo/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gLogger "gorm.io/gorm/logger"

	// import necessary for gorm to recognize the postgres driver
	_ "github.com/lib/pq"
)

type Config struct {
	Host     string `env:"DB_HOST"`
	Port     string `env:"DB_PORT"`
	User     string `env:"DB_USER"`
	Password string `env:"DB_PASSWORD"`
	Database string `env:"DB_NAME"`
	UseSSL   bool   `env:"DB_SSL_MODE"`
	LogQuery bool   `env:"DB_LOG_QUERY"`
}

func (c *Config) Validate() error {
	if c.Host == "" ||
		c.Password == "" || c.Database == "" || c.User == "" {
		return models.ErrDatabaseCredentialNotConfigured
	}
	return nil
}

func New(c *Config) (*gorm.DB, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	SSLMode := "disable"
	if c.UseSSL {
		SSLMode = "require"
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		c.Host, c.User, c.Password, c.Database, c.Port, SSLMode)

	cfg := &gorm.Config{}
	if !c.LogQuery {
		cfg.Logger = gLogger.Discard
	}

	// This handles the driver registration internally
	db, err := gorm.Open(postgres.Open(dsn), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to open gorm connection: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB from gorm: %w", err)
	}

	sqlDB.SetMaxIdleConns(50)
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetConnMaxLifetime(1 * 60 * 60)

	return db, nil
}
