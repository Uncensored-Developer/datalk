package base

import (
	"log/slog"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
)

type Base struct {
	logger *slog.Logger
	name   string
	config config.Config
}

func (b *Base) Logger() *slog.Logger {
	return b.logger
}

func (b *Base) Config() config.Config {
	return b.config
}

func NewBase(name string, logger *slog.Logger, config config.Config) *Base {
	return &Base{
		logger: logger.With("service", name),
		name:   name,
		config: config,
	}
}
