package schemas

import "time"

type EmbeddingJob struct {
	SnapshotID   int32
	Status       string
	ErrorMessage *string
	RetryCount   int32
	StartedAt    time.Time
	CompletedAt  *time.Time
}
