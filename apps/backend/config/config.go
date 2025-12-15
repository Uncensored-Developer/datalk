package config

import (
	"log"

	"github.com/caarlos0/env/v11"
)

type AppEnv string

const (
	AppEnvTest       AppEnv = "test"
	AppEnvProduction AppEnv = "production"
)

type Config struct {
	AppName string `env:"APP_NAME"`
	AppEnv  AppEnv `env:"APP_ENV"`
	ApiPort int    `env:"PORT"`

	DbDSN string `env:"DB_DSN"`
}

func MustLoad() Config {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to parse env: %v \n", err)
	}
	return cfg
}
