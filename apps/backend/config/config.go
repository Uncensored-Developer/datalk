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

	DBName     string `env:"DB_NAME"`
	DBHost     string `env:"DB_HOST"`
	DBPort     int    `env:"DB_PORT"`
	DBUser     string `env:"DB_USER"`
	DBPassword string `env:"DB_PASSWORD"`
	DBSSLMode  string `env:"DB_SSLMODE"`
}

func MustLoad() Config {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to parse env: %v \n", err)
	}
	return cfg
}
