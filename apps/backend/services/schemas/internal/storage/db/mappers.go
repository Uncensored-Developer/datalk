package db

import (
	"encoding/json"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/gotidy/ptr"
	pgvector "github.com/pgvector/pgvector-go"
	"github.com/stephenafamo/bob/types"
)

func snapshotToDB(snapshot *schemas.Snapshot) *models.SchemaSnapshotSetter {
	var introspectedAt omit.Val[time.Time]
	if !snapshot.IntrospectedAt.IsZero() {
		introspectedAt = omit.From(snapshot.IntrospectedAt)
	}

	return &models.SchemaSnapshotSetter{
		ConnectionID:   omit.From(snapshot.ConnectionID),
		NamespaceID:    omit.From(snapshot.NamespaceID),
		SchemaHash:     omit.From(snapshot.SchemaHash),
		SliceJSON:      omit.From(types.NewJSON(snapshot.SchemaJSON)),
		Status:         omit.From(string(snapshot.Status)),
		ErrorMessage:   omitnull.FromPtr(snapshot.ErrorMessage),
		IntrospectedAt: introspectedAt,
	}
}

func snapshotFromDB(dbSnapshot *models.SchemaSnapshot) (*schemas.Snapshot, error) {
	var errorMessage *string
	if msg, ok := dbSnapshot.ErrorMessage.Get(); ok {
		errorMessage = ptr.Of(msg)
	}

	return &schemas.Snapshot{
		ID:             dbSnapshot.ID,
		ConnectionID:   dbSnapshot.ConnectionID,
		NamespaceID:    dbSnapshot.NamespaceID,
		SchemaHash:     dbSnapshot.SchemaHash,
		SchemaJSON:     dbSnapshot.SliceJSON.Val,
		Status:         schemas.SnapshotStatus(dbSnapshot.Status),
		ErrorMessage:   errorMessage,
		IntrospectedAt: dbSnapshot.IntrospectedAt,
	}, nil
}

func chunkToDB(chunk *schemas.Chunk) *models.SchemaChunkSetter {
	var embedding omitnull.Val[pgvector.Vector]
	if len(chunk.Embedding) > 0 {
		embedding = omitnull.From(pgvector.NewVector(chunk.Embedding))
	}

	metadata := chunk.Metadata
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}

	return &models.SchemaChunkSetter{
		SnapshotID:   omit.From(chunk.SnapshotID),
		ConnectionID: omit.From(chunk.ConnectionID),
		NamespaceID:  omit.From(chunk.NamespaceID),
		ObjectType:   omit.From(chunk.ObjectType),
		ObjectName:   omit.From(chunk.ObjectName),
		SchemaJSON:   omit.From(types.NewJSON(chunk.SchemaJSON)),
		Content:      omit.From(chunk.Content),
		Embedding:    embedding,
		Metadata:     omit.From(types.NewJSON(json.RawMessage(metadata))),
		CreatedAt:    omit.From(chunk.CreatedAt),
	}
}

func chunkFromDB(dbChunk *models.SchemaChunk) (*schemas.Chunk, error) {
	var embedding []float32
	if v, ok := dbChunk.Embedding.Get(); ok {
		embedding = v.Slice()
	}

	return &schemas.Chunk{
		ID:           dbChunk.ID,
		SnapshotID:   dbChunk.SnapshotID,
		ConnectionID: dbChunk.ConnectionID,
		NamespaceID:  dbChunk.NamespaceID,
		ObjectType:   dbChunk.ObjectType,
		ObjectName:   dbChunk.ObjectName,
		SchemaJSON:   dbChunk.SchemaJSON.Val,
		Content:      dbChunk.Content,
		Embedding:    embedding,
		Metadata:     dbChunk.Metadata.Val,
		CreatedAt:    dbChunk.CreatedAt,
	}, nil
}
