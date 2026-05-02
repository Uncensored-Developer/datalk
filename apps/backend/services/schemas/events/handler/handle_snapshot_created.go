package handler

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pubsub"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/events"
	"github.com/mdobak/go-xerrors"
)

func (h *Handler) HandleSnapshotCreated(ctx context.Context, msg pubsub.Message) error {
	var content events.SnapshotCreatedContent
	if err := msg.DecodeBody(&content); err != nil {
		return xerrors.Newf("failed to unmarshal snapshot created content: %w", err)
	}
	return h.service.EmbedSnapshotContent(ctx, content.SnapshotID)
}
