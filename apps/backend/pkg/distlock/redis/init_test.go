package redis

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/logger"
	"github.com/go-redis/redis/v8"
)

var (
	distLocker *DistributedLocker
	cfg        config.Config
)

func setupTestRedis(cfg config.Config) (*DistributedLocker, func() error, error) {
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse redis url from config: %s", cfg.RedisURL)
	}

	client := redis.NewClient(opt)
	// Ping to ensure Redis is up
	err = client.Ping(context.Background()).Err()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	locker := NewDistributedLocker(opt)

	cleanup := func() error {
		if err := client.FlushDB(context.Background()).Err(); err != nil {
			return fmt.Errorf("failed to flush Redis DB: %w", err)
		}

		if err := client.Close(); err != nil {
			return fmt.Errorf("failed to close Redis client: %w", err)
		}
		return nil
	}
	return locker, cleanup, nil
}

func TestMain(m *testing.M) {
	cfg = config.MustLoad()

	slogLogger := logger.SetupLogger(cfg)
	locker, cleanup, err := setupTestRedis(cfg)
	if err != nil {
		slogLogger.Warn("failed to setup test redis", logger.Err(err))
		return
	}
	distLocker = locker

	result := m.Run()

	if err := cleanup(); err != nil {
		slogLogger.Error("failed to cleanup test redis", logger.Err(err))
	}

	os.Exit(result)
}
