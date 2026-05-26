# Backend Architecture

This document describes the backend architecture for Datalk. It focuses on the Go API, service boundaries, persistence model, background workflows, and the deliberate choice to rely on Postgres for most infrastructure concerns.

## Goals

Datalk is designed to be self-hosted with a small operational footprint. The backend should:

- expose a stable HTTP API for the React frontend.
- manage users, authentication, database connections, schema metadata, provider configuration, and chat conversations.
- introspect connected databases and store schema snapshots.
- embed schema context for retrieval during SQL generation.
- generate and execute SQL through configured LLM providers.
- serve the frontend and backend from one deployable binary.
- minimize required infrastructure beyond Postgres and Ollama.

## Postgres-First Design

Datalk intentionally relies on Postgres for most backend infrastructure:

- primary relational database.
- schema snapshot storage.
- vector storage and vector search through `pgvector`.
- pub/sub through Postgres `LISTEN`/`NOTIFY`.
- distributed locking through Postgres advisory locks.
- migrations through SQL migrations.

This is a deliberate product and operational choice. The goal is to reduce the number of dependencies required to set up and run the app. Instead of requiring a separate vector database, message broker, and lock service, Datalk uses Postgres for those responsibilities where it is good enough and operationally simpler.

Ollama remains required for schema embedding because it is the currently supported embedding backend. Postgres stores and searches those embeddings; Ollama produces them.

## Architecture Diagram

```text
                                  ┌────────────────────────┐
                                  │ Browser / React Web App│
                                  └───────────┬────────────┘
                                              │ HTTP
                                              ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                              Datalk Go Binary                                │
│                                                                              │
│  ┌────────────────────────┐       ┌───────────────────────────────────────┐  │
│  │ Embedded Static Assets │◄──────┤ Echo HTTP Server                      │  │
│  │ React index + assets   │       │ /api/* JSON API                       │  │
│  └────────────────────────┘       │ /* SPA fallback for non-API routes    │  │
│                                   └───────────────────┬───────────────────┘  │
│                                                       │                      │
│                                                       ▼                      │
│                                      ┌──────────────────────────────────┐    │
│                                      │ JWT Auth Middleware              │    │
│                                      │ public auth routes bypass this   │    │
│                                      └────────────────┬─────────────────┘    │
│                                                       │                      │
│        ┌──────────────────┬──────────────────┬────────┴────────┬─────────┐   │
│        ▼                  ▼                  ▼                 ▼         │   │
│ ┌──────────────┐   ┌──────────────┐   ┌──────────────┐  ┌──────────────┐ │   │
│ │ Users        │   │ Connections  │   │ Schemas      │  │ Chat         │ │   │
│ │ Service      │   │ Service      │   │ Service      │  │ Service      │ │   │
│ └──────┬───────┘   └──────┬───────┘   └──────┬───────┘  └──────┬───────┘ │   │
└────────┼──────────────────┼──────────────────┼─────────────────┼─────────┘   │
         │                  │                  │                 │             │
         │                  │                  │                 │             │
         ▼                  ▼                  ▼                 ▼             │
┌──────────────────────────────────────────────────────────────────────────────┐
│                         Postgres + pgvector                                  │
│                                                                              │
│  Primary relational data                                                     │
│  Users, refresh tokens, connections, access grants                           │
│  Schema snapshots, schema chunks, embedding jobs                             │
│  Conversations, messages, LLM calls, provider configs                        │
│  Vector search via pgvector                                                  │
│  Pub/Sub via LISTEN / NOTIFY                                                 │
│  Distributed locking via advisory locks                                      │
└──────────────────────────────────────────────────────────────────────────────┘
                                      ▲
                                      │ snapshot-created event / advisory lock
                                      │ vector store + vector search
                                      │
                ┌─────────────────────┴────────────────────┐
                │                                          │
                ▼                                          ▼
     ┌───────────────────────┐                  ┌────────────────────────┐
     │ Ollama Embedding API  │                  │ Atlas Introspector     │
     │ nomic-embed-text      │                  │ reads DB schema        │
     └───────────────────────┘                  └───────────┬────────────┘
                                                            │
                                                            ▼
                                                  ┌──────────────────────┐
                                                  │ User Databases       │
                                                  │ Postgres / MySQL     │
                                                  └──────────▲───────────┘
                                                             │
                  ┌──────────────────────────────┐           │
                  │ LLM Provider Clients         │           │
                  │ OpenAI / Anthropic / Gemini  │           │
                  │ Ollama                       │           │
                  └──────────────▲───────────────┘           │
                                 │                           │
                                 │ generate SQL              │ execute SQL
                                 │                           │
                           ┌─────┴───────────────────────────┴─────┐
                           │ Chat Service + SQL Runner             │
                           │ validates generated SQL before running│
                           └───────────────────────────────────────┘
```

## High-Level Runtime

The backend entrypoint is:

```text
apps/backend/cmd/api/main.go
```

Startup flow:

1. Load environment configuration.
2. Validate required secrets.
3. Connect to Postgres and optionally run migrations with `--try-migrate`.
4. Create shared infrastructure:
   - logger
   - Postgres distributed locker
   - Postgres pub/sub bus
5. Construct domain services:
   - users
   - connections
   - schemas
   - chat
6. Configure Echo middleware:
   - request logging
   - panic recovery
   - CORS
7. Register public auth routes under `/api`.
8. Register protected API routes under `/api`.
9. Register static frontend serving for non-API routes.
10. Subscribe schema embedding event handlers.
11. Start the HTTP server.
12. Shutdown gracefully on interrupt.

The HTTP server serves both API and frontend:

```text
/api/*      JSON API routes
/assets/*   embedded frontend assets
/*          React SPA fallback for non-API routes
```

## Authentication and Users Service

The users service owns:

- first-run setup status.
- first owner account creation.
- login.
- refresh token rotation.
- logout token revocation.
- current user lookup.
- password changes.
- admin user listing, creation, and updates.

Roles:

```text
owner
admin
member
```

`owner` and `admin` are treated as admin-level users. The owner role is special because the database enforces a single owner.

Authentication uses:

- JWT access tokens.
- opaque refresh tokens.
- hashed refresh-token storage.
- token rotation on refresh.
- Argon2 password hashing.

The backend does not use refresh-token cookies. Refresh tokens are passed in request bodies.

## Connections Service

The connections service owns:

- creating database connections.
- updating connection metadata.
- deleting connections.
- listing visible connections.
- connection access grants.
- encrypted DSN handling.

Admin access model:

- owners/admins can see and manage all connections.
- members can only see connections granted to them.

DSNs are encrypted server-side through `pkg/secrets`. Services decrypt into local copies when a downstream operation requires a real DSN. HTTP responses must not return plaintext or encrypted secrets.

Connections are also used by:

- schema refresh.
- SQL execution in chat.
- frontend connection selection.

## Schemas Service

The schemas service owns:

- database schema introspection.
- schema snapshot creation.
- snapshot change detection.
- schema text rendering.
- chunking schema text for embedding.
- embedding schema chunks through Ollama.
- storing embeddings in Postgres/pgvector.
- retrieving relevant schema context for chat.

### Refresh Flow

Schema refresh starts from:

```text
POST /api/connections/:connection_id/schema-snapshot/refresh
```

Flow:

1. Fetch the connection and decrypted DSN.
2. Use an Atlas-backed introspector to inspect the database.
3. Render the catalog into stable text.
4. Compare against previous snapshots.
5. Insert a new snapshot if the schema changed.
6. Publish a snapshot-created event through Postgres pub/sub.

### Embedding Flow

The schemas service subscribes to snapshot-created events at API startup. When an event is received:

1. A Postgres distributed lock prevents concurrent duplicate embedding work.
2. The snapshot text is split into chunks.
3. Chunks are embedded with Ollama.
4. Embeddings are stored in `schema_chunks`.
5. Embedding job status is updated.

Vector search uses pgvector in Postgres. This avoids requiring a separate vector database.

### Retrieval Flow

When chat needs schema context:

1. The user question is embedded.
2. pgvector search finds relevant schema chunks for the selected connection.
3. Distinct chunks are returned to the chat service.
4. The chat service uses that context during SQL generation.

## Chat Service

The chat service owns:

- conversation creation and listing.
- message creation and listing.
- provider config save/list.
- provider model discovery.
- SQL generation.
- SQL validation.
- SQL execution.
- storing LLM call metadata.

Chat depends on:

- users service for user context.
- connections service for connection access and decrypted DSNs.
- schemas service for retrieved schema context.
- LLM clients for generation.

Supported LLM client packages include:

```text
openai
anthropic
gemini
ollama
```

Provider configs are provider-unique and editable. They are not deletable because historical chat and LLM call records may reference provider configurations.

### Message Send Flow

For a message request:

1. Validate the conversation and user access.
2. Store the user message.
3. Retrieve relevant schema chunks for the selected connection.
4. Build SQL-generation context.
5. Resolve the configured provider/model.
6. Call the LLM provider.
7. Validate generated SQL for the connection kind.
8. Execute SQL through the SQL runner.
9. Store assistant message, model call, SQL, and execution result metadata.
10. Return the response to the frontend.

The current design focuses message responses on conversation/message/model-call details. Schema chunks are not returned by default.

## LLM Provider Layer

The LLM provider layer hides provider-specific APIs behind common interfaces.

Responsibilities:

- normalize model listing.
- normalize SQL-generation calls.
- validate structured provider responses.
- redact secrets from logs/errors.
- allow tests to mock provider behavior.

Provider-specific behavior stays under:

```text
services/chat/internal/chat/llm/<provider>
```

This keeps the chat orchestration code provider-agnostic.

## SQL Runner

The SQL runner executes generated SQL against configured database connections.

It includes validators for supported database kinds, currently including Postgres and MySQL-specific validators.

Validation exists because generated SQL should be constrained before execution. The service should reject unsupported, unsafe, or invalid SQL before it reaches a user database.

## Static Web Serving

The backend embeds frontend assets through Go `embed`:

```text
servers/echo/staticweb
```

Behavior:

- direct static assets are served from embedded files.
- non-API paths return `index.html` for React Router support.
- `/api` and `/api/*` are never handled by the SPA fallback.

The production build pipeline:

1. build `apps/web`.
2. copy `apps/web/dist` into `apps/backend/servers/echo/staticweb/dist`.
3. compile the backend binary.

This produces one binary that serves both API and frontend.

## Database and Migrations

Migrations live in:

```text
apps/backend/db/migrations
```

The database contains tables for:

- users and auth fields.
- refresh tokens.
- organizations.
- connections.
- connection access.
- schema snapshots.
- schema chunks with pgvector embeddings.
- schema embedding jobs.
- chat conversations.
- chat messages.
- LLM calls.
- provider configs.

Generated Bob models live under:

```text
apps/backend/db/models
apps/backend/db/info
apps/backend/db/errors
apps/backend/db/factory
```

Run model generation after migrations that alter generated database shape.

## Pub/Sub

The pub/sub abstraction lives in:

```text
apps/backend/pkg/pubsub
```

Implementations include:

- in-memory pub/sub for tests.
- Postgres pub/sub for runtime.

The runtime bus is backed by Postgres `LISTEN`/`NOTIFY`. This is used for schema snapshot embedding events.

Using Postgres here avoids adding a message broker for the current background workflow needs.

## Distributed Locks

Distributed locking lives in:

```text
apps/backend/pkg/distlock
```

Implementations include:

- dummy lock implementation.
- Redis lock implementation.
- Postgres advisory-lock implementation.

The runtime API uses the Postgres implementation. The main use case is preventing duplicate schema embedding work when multiple app instances receive the same event.

Again, this keeps the deployment simpler by using Postgres instead of requiring a separate lock service.

## Secrets and Encryption

Secret handling lives in:

```text
apps/backend/pkg/secrets
```

Encrypted values include:

- database connection DSNs.
- provider API keys.

Encryption uses AES-GCM with `PROVIDER_CONFIG_SECRET`. Code should decrypt into local service-level copies only when needed and must not return secret values in API responses.

## Error Handling

Domain services return typed domain errors from package-specific `pkg/errors` packages.

HTTP handlers map those errors to response status codes:

- `400` for validation or invalid request errors.
- `401` for missing/invalid authentication.
- `403` for forbidden or inactive users.
- `404` for missing resources.
- `409` for setup conflicts.
- `500` for internal errors with sanitized response bodies.

This keeps internal error details out of API responses while preserving useful logs.

## Testing Strategy

Testing is split by layer:

- service tests validate business logic with mocks.
- storage tests validate Postgres behavior and migrations.
- handler tests validate HTTP request/response mapping.
- provider client tests validate provider-specific request/response handling.
- staticweb tests validate SPA fallback and API exclusion.

DB-dependent tests use `db/common.TestRunner`, create isolated test schemas, and run migrations.

Generated mocks are produced with mockery through `go generate`.

## Operational Dependencies

Required:

- Datalk API binary.
- Postgres with pgvector.
- Ollama with `nomic-embed-text`.

Optional or situational:

- Remote LLM provider APIs for non-Ollama chat generation.
- Redis remains in some package code/tests, but the default runtime path uses Postgres for locks and pub/sub.

The central architecture principle is to keep the minimum production deployment small: one app binary, Postgres, and Ollama.

## Known Tradeoffs

Postgres-first infrastructure keeps setup simple, but it has tradeoffs:

- Postgres pub/sub is not a full message queue.
- pgvector is not a dedicated vector database.
- advisory locks require careful resource naming and connection handling.
- long-running or high-volume background jobs may eventually need a dedicated queue.
- very large embedding/search workloads may eventually justify a separate vector store.

Those tradeoffs are acceptable for the current product because they make the self-hosted setup much easier and keep the operational surface area small.
