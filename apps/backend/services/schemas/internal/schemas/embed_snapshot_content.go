package schemas

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
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
	maxChunkContentRunes      = 6000
	splitChunkHeaderReserve   = 256
	minSplitChunkBodyRunes    = 1
)

type chunkMetadata struct {
	Namespace     string `json:"namespace"`
	QualifiedName string `json:"qualified_name"`
	Source        string `json:"source"`
	Model         string `json:"model"`
	SchemaHash    string `json:"schema_hash,omitempty"`
	ChunkPart     int    `json:"chunk_part,omitempty"`
	ChunkTotal    int    `json:"chunk_total,omitempty"`
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
	startedAt := time.Now()
	defer func() {
		if retErr == nil {
			return
		}

		s.Logger().Warn(
			"schema snapshot embedding failed",
			slog.Any("err", retErr),
			slog.Int("snapshot_id", int(snapshotID)),
			slog.Int("latency_ms", int(time.Since(startedAt).Milliseconds())),
		)
	}()

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

	s.Logger().Info(
		"schema snapshot embedded",
		slog.Int("snapshot_id", int(snapshotID)),
		slog.Int("connection_id", int(snapshot.ConnectionID)),
		slog.Int("chunks", len(chunks)),
		slog.Int("latency_ms", int(time.Since(startedAt).Milliseconds())),
	)

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
			tableChunks, err := buildTableChunks(snapshot, catalog.Kind, namespace.Name, &table)
			if err != nil {
				return nil, err
			}
			chunks = append(chunks, tableChunks...)
		}

		for _, view := range utils.SortedViews(namespace.Views) {
			viewChunks, err := buildViewChunks(snapshot, catalog.Kind, namespace.Name, &view)
			if err != nil {
				return nil, err
			}
			chunks = append(chunks, viewChunks...)
		}
	}

	return chunks, nil
}

func buildTableChunks(snapshot *schematypes.Snapshot, kind introspector.DBKind, namespace string, table *introspector.Table) ([]*schematypes.Chunk, error) {
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

	return buildObjectChunks(snapshot, namespace, qualifiedName, chunkObjectTypeTable, schemaJSON, content)
}

func buildViewChunks(snapshot *schematypes.Snapshot, kind introspector.DBKind, namespace string, view *introspector.View) ([]*schematypes.Chunk, error) {
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

	return buildObjectChunks(snapshot, namespace, qualifiedName, chunkObjectTypeView, schemaJSON, content)
}

func buildObjectChunks(
	snapshot *schematypes.Snapshot,
	namespace string,
	qualifiedName string,
	objectType string,
	schemaJSON []byte,
	content string,
) ([]*schematypes.Chunk, error) {
	contentParts := splitChunkContent(content, objectType, qualifiedName)
	chunks := make([]*schematypes.Chunk, 0, len(contentParts))
	now := time.Now().UTC()
	for i, part := range contentParts {
		metadataPayload := chunkMetadata{
			Namespace:     namespace,
			QualifiedName: qualifiedName,
			Source:        chunkSourceCatalogText,
			Model:         embeddingollama.ModelName,
			SchemaHash:    snapshot.SchemaHash,
		}
		if len(contentParts) > 1 {
			metadataPayload.ChunkPart = i + 1
			metadataPayload.ChunkTotal = len(contentParts)
		}

		metadata, err := json.Marshal(metadataPayload)
		if err != nil {
			return nil, xerrors.Newf("failed to marshal %s chunk metadata: %w", objectType, err)
		}

		chunks = append(chunks, &schematypes.Chunk{
			SnapshotID:   snapshot.ID,
			ConnectionID: snapshot.ConnectionID,
			ObjectType:   objectType,
			ObjectName:   qualifiedName,
			SchemaJSON:   schemaJSON,
			Content:      part,
			Metadata:     metadata,
			CreatedAt:    now,
		})
	}

	return chunks, nil
}

func splitChunkContent(content string, objectType string, qualifiedName string) []string {
	if runeLen(content) <= maxChunkContentRunes {
		return []string{content}
	}

	bodyLimit := maxChunkContentRunes - splitChunkHeaderReserve
	if bodyLimit <= 0 {
		bodyLimit = minSplitChunkBodyRunes
	}

	var bodies []string
	for {
		bodies = splitContentIntoBodies(content, bodyLimit)
		nextBodyLimit := maxChunkContentRunes - maxChunkPartHeaderRunes(objectType, qualifiedName, len(bodies))
		if nextBodyLimit < minSplitChunkBodyRunes {
			nextBodyLimit = minSplitChunkBodyRunes
		}
		if nextBodyLimit >= bodyLimit {
			break
		}
		bodyLimit = nextBodyLimit
	}

	parts := make([]string, 0, len(bodies))
	for i, body := range bodies {
		header := chunkPartHeader(objectType, qualifiedName, i+1, len(bodies))
		parts = append(parts, header+body)
	}

	return parts
}

func splitContentIntoBodies(content string, maxBodyRunes int) []string {
	if content == "" {
		return []string{""}
	}

	parts := make([]string, 0)
	var current strings.Builder
	currentRunes := 0

	flush := func() {
		if currentRunes == 0 {
			return
		}
		parts = append(parts, current.String())
		current.Reset()
		currentRunes = 0
	}

	for i, line := range strings.Split(content, "\n") {
		lineContent := line
		if i > 0 {
			lineContent = "\n" + lineContent
		}
		lineRunes := runeLen(lineContent)
		if lineRunes > maxBodyRunes {
			flush()
			for _, piece := range splitRunes(lineContent, maxBodyRunes) {
				parts = append(parts, piece)
			}
			continue
		}

		if currentRunes > 0 && currentRunes+lineRunes > maxBodyRunes {
			flush()
		}
		current.WriteString(lineContent)
		currentRunes += lineRunes
	}
	flush()

	if len(parts) == 0 {
		return []string{""}
	}
	return parts
}

func splitRunes(value string, limit int) []string {
	if limit <= 0 {
		return []string{value}
	}

	valueRunes := []rune(value)
	parts := make([]string, 0, (len(valueRunes)/limit)+1)
	for start := 0; start < len(valueRunes); start += limit {
		end := start + limit
		if end > len(valueRunes) {
			end = len(valueRunes)
		}
		parts = append(parts, string(valueRunes[start:end]))
	}
	return parts
}

func chunkPartHeader(objectType string, qualifiedName string, part int, total int) string {
	return fmt.Sprintf("object_type: %s\nqualified_name: %s\nchunk_part: %d/%d\n\n", objectType, qualifiedName, part, total)
}

func maxChunkPartHeaderRunes(objectType string, qualifiedName string, total int) int {
	return runeLen(chunkPartHeader(objectType, qualifiedName, total, total))
}

func runeLen(value string) int {
	return len([]rune(value))
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
