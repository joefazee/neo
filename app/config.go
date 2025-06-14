package app

import (
	"github.com/joefazee/neo/app/database"
	"github.com/joefazee/neo/app/markets"
	"github.com/joefazee/neo/app/prediction"
	"github.com/joefazee/neo/app/user"
	"github.com/joefazee/neo/internal/nexus"
)

type Config struct {
	DB         database.Config
	User       user.Config
	Market     markets.Config
	Prediction prediction.Config

	AppHost string `env:"APP_HOST" default:"localhost"`
	AppPort string `env:"APP_PORT" default:"8080"`
	Env     string `env:"APP_ENV" default:"development"`
}

// LoadConfig loads the application configuration from environment variables or a config file.
func LoadConfig() (*Config, error) {
	c := &Config{}
	err := nexus.NewLoader().Load(c)
	return c, err
}
