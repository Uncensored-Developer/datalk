package schemas

import (
	"encoding/json"
	"time"
)

type Chunk struct {
	ID           int64
	SnapshotID   int32
	ConnectionID int32
	ObjectType   string
	ObjectName   string
	SchemaJSON   json.RawMessage
	Content      string
	Embedding    []float32
	Metadata     []byte
	CreatedAt    time.Time
}
