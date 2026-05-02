package schemas

import "time"

const (
	EmbeddingJobStatusPending    = "PENDING"
	EmbeddingJobStatusProcessing = "PROCESSING"
	EmbeddingJobStatusCompleted  = "COMPLETED"
	EmbeddingJobStatusFailed     = "FAILED"
)

type EmbeddingJob struct {
	SnapshotID   int32
	Status       string
	ErrorMessage *string
	RetryCount   int32
	StartedAt    time.Time
	CompletedAt  *time.Time
}
