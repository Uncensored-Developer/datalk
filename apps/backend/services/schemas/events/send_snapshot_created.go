package events

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pubsub"
)

type SnapshotCreatedContent struct {
	SnapshotID int32 `json:"snapshot_id"`
}

func SendSnapshotCreated(ctx context.Context, content SnapshotCreatedContent) error {
	msg, err := pubsub.NewMessage(SnapshotCreated, "", content)
	if err != nil {
		return err
	}
	return pubsub.Send(ctx, msg)
}
