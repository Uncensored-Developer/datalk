package schemas

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/distlock"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pubsub"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/api"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/events"
	eventhandler "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/events/handler"
	internalschemas "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/embedding"
	embeddingollama "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/embedding/ollama"
	"github.com/mdobak/go-xerrors"
)

type Schemas struct {
	API     *api.Api
	Service *internalschemas.Service
	Handler *eventhandler.Handler
}

func New(ctx context.Context, conn *sql.DB, cfg config.Config, logger *slog.Logger, locker distlock.DistributedLocker, connectionGetter internalschemas.ConnectionGetter) (*Schemas, error) {
	var embeddingClient embedding.Client
	if cfg.EmbeddingEnabled {
		ollamaClient, err := embeddingollama.NewClient(cfg.OllamaBaseURL, &http.Client{Timeout: cfg.EmbeddingTimeout})
		if err != nil {
			return nil, xerrors.Newf("failed to create ollama embedding client: %w", err)
		}

		checkCtx, cancel := context.WithTimeout(ctx, cfg.EmbeddingTimeout)
		defer cancel()
		if err := ollamaClient.Check(checkCtx); err != nil {
			return nil, xerrors.Newf("ollama embedding preflight failed: %w", err)
		}

		embeddingClient = ollamaClient
	}

	schemasService := internalschemas.NewService(
		conn,
		cfg,
		logger,
		connectionGetter,
		newIntrospectorFactory(),
		locker,
		embeddingClient,
	)

	return &Schemas{
		API:     api.New(logger, cfg, schemasService),
		Service: schemasService,
		Handler: eventhandler.New(logger, cfg, schemasService),
	}, nil
}

func (s *Schemas) SubscribeSnapshotEmbedding(ctx context.Context, subscriber pubsub.Subscriber) (pubsub.Subscription, error) {
	if s == nil || s.Service == nil || !s.Service.Config().EmbeddingEnabled {
		return nil, nil
	}

	return subscriber.Subscribe(
		ctx,
		events.SnapshotCreated,
		s.Handler,
		pubsub.WithGroup("schemas-snapshot-embedding"),
		pubsub.WithConcurrency(s.Service.Config().EmbeddingConcurrency),
		pubsub.WithMaxRetries(s.Service.Config().EmbeddingMaxRetries),
		pubsub.WithRetryBackoff(time.Second),
	)
}
