package schemas

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/distlock"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/catalogtext"
	embeddingollama "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/embedding/ollama"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/introspector"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/utils"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/errors"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/gotidy/ptr"
	"github.com/mdobak/go-xerrors"
)

const (
	defaultEmbeddingBatchSize = 16
	chunkObjectTypeTable      = "table"
	chunkObjectTypeView       = "view"
	chunkSourceCatalogText    = "catalogtext"
)

type chunkMetadata struct {
	Namespace     string `json:"namespace"`
	QualifiedName string `json:"qualified_name"`
	Source        string `json:"source"`
	Model         string `json:"model"`
	SchemaHash    string `json:"schema_hash,omitempty"`
}

type chunkSchema struct {
	DatabaseKind  introspector.DBKind `json:"database_kind"`
	Namespace     string              `json:"namespace"`
	ObjectType    string              `json:"object_type"`
	ObjectName    string              `json:"object_name"`
	QualifiedName string              `json:"qualified_name"`
	Table         *introspector.Table `json:"table,omitempty"`
	View          *introspector.View  `json:"view,omitempty"`
}

func (s *Service) EmbedSnapshotContent(ctx context.Context, snapshotID int32) (retErr error) {
	// Embedding is optional at the service level, so fail early when the feature
	// is disabled or the runtime dependency was not wired in.
	if !s.Config().EmbeddingEnabled {
		return errors.ErrEmbeddingDisabled
	}
	if s.embeddingClient == nil {
		return xerrors.New("embedding client is not configured")
	}

	snapshots, err := s.storage.ListSnapshots(ctx, storage.SnapshotsFilter{ID: []int32{snapshotID}})
	if err != nil {
		return xerrors.Newf("failed to list snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		return storage.ErrSnapshotNotFound
	}
	snapshot := snapshots[0]

	// Serialize work per connection so concurrent snapshot refreshes or retries
	// cannot replace the same chunk set underneath each other.
	key := fmt.Sprintf("schemas-core:embed-snapshot-content:%d", snapshot.ConnectionID)
	lock, err := s.locker.Lock(ctx, []string{key}, distlock.LockOptions{Wait: true})
	if err != nil {
		return xerrors.Newf("failed to acquire lock: %w", err)
	}
	defer lock.Unlock()

	// Move the embedding job into PROCESSING before any expensive work starts so
	// retries and operators can observe in-flight state immediately.
	existingJob, err := s.storage.GetEmbeddingJob(ctx, snapshotID)
	if err != nil {
		return xerrors.Newf("failed to get embedding job: %w", err)
	}

	processingJob := embeddingJobForProcessing(snapshotID, existingJob)
	if err := s.storage.UpsertEmbeddingJob(ctx, processingJob); err != nil {
		return xerrors.Newf("failed to set embedding job processing: %w", err)
	}

	// Any failure after the PROCESSING transition should leave behind a FAILED
	// job row with the latest error so retries have correct state to build on.
	defer func() {
		if retErr == nil {
			return
		}

		failedJob := embeddingJobForFailure(processingJob, retErr.Error())
		if err := s.storage.UpsertEmbeddingJob(ctx, failedJob); err != nil {
			retErr = xerrors.Newf("%w; failed to update embedding job: %v", retErr, err)
		}
	}()

	// The snapshot JSON is the canonical introspection artifact; chunk rendering
	// and embedding are derived from that payload on every run.
	var catalog introspector.Catalog
	if err := json.Unmarshal(snapshot.SchemaJSON, &catalog); err != nil {
		return xerrors.Newf("failed to unmarshal snapshot schema json: %w", err)
	}

	// Rebuild the full per-object chunk set, embed each chunk, then replace the
	// persisted rows in one pass so reruns stay deterministic and idempotent.
	chunks, err := buildSnapshotChunks(snapshot, &catalog)
	if err != nil {
		return xerrors.Newf("failed to build snapshot chunks: %w", err)
	}

	if err := s.embedChunks(ctx, chunks); err != nil {
		return xerrors.Newf("failed to embed chunks: %w", err)
	}

	if err := s.storage.ReplaceChunks(ctx, snapshot.ID, chunks); err != nil {
		return xerrors.Newf("failed to replace snapshot chunks: %w", err)
	}

	// Mark the job complete only after the new chunk set has been persisted.
	completedJob := embeddingJobForCompletion(snapshotID, processingJob)
	if err := s.storage.UpsertEmbeddingJob(ctx, completedJob); err != nil {
		return xerrors.Newf("failed to set embedding job completed: %w", err)
	}

	return nil
}

func (s *Service) embedChunks(ctx context.Context, chunks []*schematypes.Chunk) error {
	batchSize := s.Config().EmbeddingBatchSize
	if batchSize <= 0 {
		batchSize = defaultEmbeddingBatchSize
	}

	for start := 0; start < len(chunks); start += batchSize {
		end := start + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		if err := s.embedChunkBatch(ctx, chunks[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) embedChunkBatch(ctx context.Context, batch []*schematypes.Chunk) error {
	contents := make([]string, 0, len(batch))
	for _, chunk := range batch {
		contents = append(contents, chunk.Content)
	}

	embeddings, err := s.embeddingClient.EmbedTexts(ctx, contents)
	if err != nil {
		return err
	}
	if len(embeddings) != len(batch) {
		return xerrors.Newf("embedding count mismatch: expected %d, got %d", len(batch), len(embeddings))
	}
	for i := range batch {
		batch[i].Embedding = embeddings[i]
	}
	return nil
}

func buildSnapshotChunks(snapshot *schematypes.Snapshot, catalog *introspector.Catalog) ([]*schematypes.Chunk, error) {
	chunks := make([]*schematypes.Chunk, 0)
	for _, namespace := range utils.SortedNamespaces(catalog.Namespaces) {
		for _, table := range utils.SortedTables(namespace.Tables) {
			chunk, err := buildTableChunk(snapshot, catalog.Kind, namespace.Name, &table)
			if err != nil {
				return nil, err
			}
			chunks = append(chunks, chunk)
		}

		for _, view := range utils.SortedViews(namespace.Views) {
			chunk, err := buildViewChunk(snapshot, catalog.Kind, namespace.Name, &view)
			if err != nil {
				return nil, err
			}
			chunks = append(chunks, chunk)
		}
	}

	return chunks, nil
}

func buildTableChunk(snapshot *schematypes.Snapshot, kind introspector.DBKind, namespace string, table *introspector.Table) (*schematypes.Chunk, error) {
	qualifiedName := embeddingollama.QualifiedName(namespace, table.Name)
	content := catalogtext.RenderTable(kind, namespace, table)

	schemaJSON, err := json.Marshal(chunkSchema{
		DatabaseKind:  kind,
		Namespace:     namespace,
		ObjectType:    chunkObjectTypeTable,
		ObjectName:    table.Name,
		QualifiedName: qualifiedName,
		Table:         table,
	})
	if err != nil {
		return nil, xerrors.Newf("failed to marshal table chunk schema json: %w", err)
	}

	return buildObjectChunk(snapshot, namespace, qualifiedName, chunkObjectTypeTable, schemaJSON, content)
}

func buildViewChunk(snapshot *schematypes.Snapshot, kind introspector.DBKind, namespace string, view *introspector.View) (*schematypes.Chunk, error) {
	qualifiedName := embeddingollama.QualifiedName(namespace, view.Name)
	content := catalogtext.RenderView(kind, namespace, view)

	schemaJSON, err := json.Marshal(chunkSchema{
		DatabaseKind:  kind,
		Namespace:     namespace,
		ObjectType:    chunkObjectTypeView,
		ObjectName:    view.Name,
		QualifiedName: qualifiedName,
		View:          view,
	})
	if err != nil {
		return nil, xerrors.Newf("failed to marshal view chunk schema json: %w", err)
	}

	return buildObjectChunk(snapshot, namespace, qualifiedName, chunkObjectTypeView, schemaJSON, content)
}

func buildObjectChunk(
	snapshot *schematypes.Snapshot,
	namespace string,
	qualifiedName string,
	objectType string,
	schemaJSON []byte,
	content string,
) (*schematypes.Chunk, error) {
	metadata, err := json.Marshal(chunkMetadata{
		Namespace:     namespace,
		QualifiedName: qualifiedName,
		Source:        chunkSourceCatalogText,
		Model:         embeddingollama.ModelName,
		SchemaHash:    snapshot.SchemaHash,
	})
	if err != nil {
		return nil, xerrors.Newf("failed to marshal %s chunk metadata: %w", objectType, err)
	}

	return &schematypes.Chunk{
		SnapshotID:   snapshot.ID,
		ConnectionID: snapshot.ConnectionID,
		ObjectType:   objectType,
		ObjectName:   qualifiedName,
		SchemaJSON:   schemaJSON,
		Content:      content,
		Metadata:     metadata,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

func embeddingJobForProcessing(snapshotID int32, existing *schematypes.EmbeddingJob) *schematypes.EmbeddingJob {
	retryCount := int32(0)
	if existing != nil {
		retryCount = existing.RetryCount
	}

	return &schematypes.EmbeddingJob{
		SnapshotID: snapshotID,
		Status:     schematypes.EmbeddingJobStatusProcessing,
		RetryCount: retryCount,
		StartedAt:  time.Now().UTC(),
	}
}

func embeddingJobForCompletion(snapshotID int32, processing *schematypes.EmbeddingJob) *schematypes.EmbeddingJob {
	return &schematypes.EmbeddingJob{
		SnapshotID:  snapshotID,
		Status:      schematypes.EmbeddingJobStatusCompleted,
		RetryCount:  processing.RetryCount,
		StartedAt:   processing.StartedAt,
		CompletedAt: ptr.Of(time.Now().UTC()),
	}
}

func embeddingJobForFailure(processing *schematypes.EmbeddingJob, message string) *schematypes.EmbeddingJob {
	return &schematypes.EmbeddingJob{
		SnapshotID:   processing.SnapshotID,
		Status:       schematypes.EmbeddingJobStatusFailed,
		RetryCount:   processing.RetryCount + 1,
		StartedAt:    processing.StartedAt,
		CompletedAt:  ptr.Of(time.Now().UTC()),
		ErrorMessage: &message,
	}
}
