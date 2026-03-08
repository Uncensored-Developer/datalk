package schemas

import (
	"encoding/json"
	"time"
)

type SnapshotStatus string

const (
	SnapshotStatusStarted   SnapshotStatus = "started"
	SnapshotStatusCompleted SnapshotStatus = "completed"
	SnapshotStatusFailed    SnapshotStatus = "failed"
)

type Snapshot struct {
	ID             int32
	ConnectionID   int32
	NamespaceID    int32
	SchemaHash     string
	SchemaJSON     json.RawMessage
	Status         SnapshotStatus
	ErrorMessage   *string
	IntrospectedAt time.Time
}
