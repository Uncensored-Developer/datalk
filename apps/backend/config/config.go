package config

import (
	"log"
	"time"

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

	DBName         string `env:"DB_NAME"`
	DBHost         string `env:"DB_HOST"`
	DBPort         int    `env:"DB_PORT"`
	DBUser         string `env:"DB_USER"`
	DBPassword     string `env:"DB_PASSWORD"`
	DBSSLMode      string `env:"DB_SSLMODE"`
	DBSchema       string `env:"DB_SCHEMA"`
	GoMigrateTable string `env:"GO_MIGRATE_TABLE"`
	RedisURL       string `env:"REDIS_URL"`

	JWTSecret     string        `env:"JWT_SECRET"`
	JWTAccessTTL  time.Duration `env:"JWT_ACCESS_TTL" envDefault:"15m"`
	JWTRefreshTTL time.Duration `env:"JWT_REFRESH_TTL" envDefault:"720h"`
	JWTIssuer     string        `env:"JWT_ISSUER"`

	ProviderConfigSecret string `env:"PROVIDER_CONFIG_SECRET"`

	EmbeddingEnabled     bool          `env:"EMBEDDING_ENABLED" envDefault:"true"`
	OllamaBaseURL        string        `env:"OLLAMA_BASE_URL" envDefault:"http://localhost:11434"`
	EmbeddingBatchSize   int           `env:"EMBEDDING_BATCH_SIZE" envDefault:"16"`
	EmbeddingTimeout     time.Duration `env:"EMBEDDING_TIMEOUT" envDefault:"30s"`
	EmbeddingMaxRetries  int           `env:"EMBEDDING_MAX_RETRIES" envDefault:"3"`
	EmbeddingConcurrency int           `env:"EMBEDDING_CONCURRENCY" envDefault:"1"`
}

func MustLoad() Config {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to parse env: %v \n", err)
	}
	return cfg
}
