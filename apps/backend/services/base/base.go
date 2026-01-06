package base

import "log/slog"

type Base struct {
	logger *slog.Logger
	name   string
}

func (b *Base) Logger() *slog.Logger {
	return b.logger
}

func NewBase(name string, logger *slog.Logger) *Base {
	return &Base{
		logger: logger.With("service", name),
		name:   name,
	}
}
