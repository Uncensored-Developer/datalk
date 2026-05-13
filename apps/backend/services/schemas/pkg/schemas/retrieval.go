package schemas

import (
	"encoding/json"
	"time"
)

type RetrieveRelevantSchemaContextParams struct {
	ConnectionID        int32
	QueryText           string
	Limit               int
	SimilarityThreshold *float32
}

type RetrievedSchemaContext struct {
	ConnectionID   int32
	SnapshotID     int32
	EmbeddingModel string
	QueryText      string
	Chunks         []RetrievedChunk
	RetrievedAt    time.Time
}

type RetrievedChunk struct {
	ChunkID    int64
	ObjectType string
	ObjectName string
	Content    string
	SchemaJSON json.RawMessage
	Similarity float32
}
