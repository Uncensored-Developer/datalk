package handler

import (
	"context"
	"log/slog"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pubsub"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/events"
	"github.com/mdobak/go-xerrors"
)

type EmbedSnapshotService interface {
	EmbedSnapshotContent(ctx context.Context, snapshotID int32) error
}

type Handler struct {
	*base.Base

	service EmbedSnapshotService
}

func New(logger *slog.Logger, cfg config.Config, service EmbedSnapshotService) *Handler {
	return &Handler{
		Base:    base.NewBase("schemas-events-handler", logger, cfg),
		service: service,
	}
}

func (h *Handler) Handle(ctx context.Context, msg pubsub.Message) error {
	switch msg.Topic {
	case events.SnapshotCreated:
		return h.HandleSnapshotCreated(ctx, msg)
	default:
		return xerrors.Newf("unsupported schemas event topic: %s", msg.Topic)
	}
}
