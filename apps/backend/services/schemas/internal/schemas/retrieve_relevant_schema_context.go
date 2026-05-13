package schemas

import (
	"context"
	"strings"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ordering"
	embeddingollama "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/embedding/ollama"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage"
	schemaerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/errors"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/mdobak/go-xerrors"
)

const (
	// Eight chunks is a conservative default for schema RAG: usually enough to cover
	// the primary tables plus a couple of supporting relations without bloating the prompt.
	defaultRetrievalLimit = 8
	// We intentionally over-fetch before deduping because vector search often returns
	// several chunks for the same object; pulling 2x gives the deduper room to keep
	// `limit` distinct schema objects without a second DB round-trip.
	retrievalSearchMultiple = 2
)

func (s *Service) RetrieveRelevantSchemaContext(ctx context.Context, params schemas.RetrieveRelevantSchemaContextParams) (*schemas.RetrievedSchemaContext, error) {
	if err := s.validateRetrieveRelevantSchemaContextParams(params); err != nil {
		return nil, err
	}

	limit := resolveRetrievalLimit(params.Limit)

	snapshot, err := s.latestEmbeddedSnapshot(ctx, params.ConnectionID)
	if err != nil {
		return nil, err
	}

	queryEmbedding, err := s.embedSingleQuery(ctx, params.QueryText)
	if err != nil {
		return nil, err
	}

	retrievedChunks, err := s.storage.SearchChunks(ctx, storage.SearchChunksParams{
		SnapshotID:          snapshot.ID,
		QueryEmbedding:      queryEmbedding,
		Limit:               limit * retrievalSearchMultiple,
		SimilarityThreshold: params.SimilarityThreshold,
	})
	if err != nil {
		return nil, xerrors.Newf("failed to search schema chunks: %w", err)
	}

	chunks := dedupeRetrievedChunks(retrievedChunks, limit)
	if len(chunks) == 0 {
		return nil, schemaerrors.ErrEmbeddedSnapshotNotReady
	}

	return &schemas.RetrievedSchemaContext{
		ConnectionID:   params.ConnectionID,
		SnapshotID:     snapshot.ID,
		EmbeddingModel: embeddingollama.ModelName,
		QueryText:      params.QueryText,
		Chunks:         chunks,
		RetrievedAt:    time.Now().UTC(),
	}, nil
}

func (s *Service) validateRetrieveRelevantSchemaContextParams(params schemas.RetrieveRelevantSchemaContextParams) error {
	if strings.TrimSpace(params.QueryText) == "" {
		return xerrors.New("query text cannot be empty")
	}
	if !s.Config().EmbeddingEnabled {
		return schemaerrors.ErrEmbeddingDisabled
	}
	if s.embeddingClient == nil {
		return xerrors.New("embedding client is not configured")
	}

	return nil
}

func resolveRetrievalLimit(limit int) int {
	if limit <= 0 {
		return defaultRetrievalLimit
	}

	return limit
}

func (s *Service) embedSingleQuery(ctx context.Context, queryText string) ([]float32, error) {
	// Retrieval currently embeds exactly one query string, so we expect one vector
	// back even though the embedding client is batch-oriented.
	embeddings, err := s.embeddingClient.EmbedTexts(ctx, []string{queryText})
	if err != nil {
		return nil, xerrors.Newf("failed to embed retrieval query: %w", err)
	}
	if len(embeddings) != 1 {
		return nil, xerrors.Newf("unexpected retrieval embedding count: expected 1, got %d", len(embeddings))
	}

	return embeddings[0], nil
}

func (s *Service) latestEmbeddedSnapshot(ctx context.Context, connectionID int32) (*schemas.Snapshot, error) {
	snapshots, err := s.storage.ListSnapshots(ctx, storage.SnapshotsFilter{
		ConnectionID: []int32{connectionID},
		Ordering: ordering.Orderings[storage.SnapshotOrdering]{
			// Retrieval should prefer the newest schema view available for the connection.
			ordering.OrderByDesc(storage.SnapshotOrderingIntrospectedAt),
			ordering.OrderByDesc(storage.SnapshotOrderingID),
		},
	})
	if err != nil {
		return nil, xerrors.Newf("failed to list snapshots: %w", err)
	}
	if len(snapshots) == 0 {
		return nil, schemaerrors.ErrSnapshotNotFound
	}

	for _, snapshot := range snapshots {
		job, err := s.storage.GetEmbeddingJob(ctx, snapshot.ID)
		if err != nil {
			return nil, xerrors.Newf("failed to get embedding job: %w", err)
		}
		if job != nil && job.Status == schemas.EmbeddingJobStatusCompleted {
			return snapshot, nil
		}
	}

	return nil, schemaerrors.ErrEmbeddedSnapshotNotReady
}

func dedupeRetrievedChunks(chunks []*schemas.RetrievedChunk, limit int) []schemas.RetrievedChunk {
	if limit <= 0 {
		return []schemas.RetrievedChunk{}
	}

	seen := make(map[string]struct{}, len(chunks))
	deduped := make([]schemas.RetrievedChunk, 0, min(limit, len(chunks)))
	for _, chunk := range chunks {
		if chunk == nil {
			continue
		}
		// ObjectName is the dedupe key because multiple chunks for the same table/view
		// add less prompt value than covering another relevant schema object.
		if _, ok := seen[chunk.ObjectName]; ok {
			continue
		}
		seen[chunk.ObjectName] = struct{}{}
		deduped = append(deduped, *chunk)
		if len(deduped) == limit {
			break
		}
	}

	return deduped
}
