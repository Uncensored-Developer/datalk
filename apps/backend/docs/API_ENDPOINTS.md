# Datalk Backend API Reference

This document describes the Echo HTTP API implemented by `apps/backend`. It is intended for frontend agents building against the current backend.

## Base URL

Use the backend origin plus the `/api` prefix.

Local example:

```text
http://localhost:8080/api
```

The actual port is controlled by `PORT`.

## Authentication

Most endpoints require a JWT access token:

```http
Authorization: Bearer <access_token>
```

The access token is returned by setup, login, and refresh endpoints. Refresh tokens are sent in request bodies, not cookies.

Public endpoints:

- `POST /auth/setup`
- `POST /auth/login`
- `POST /auth/refresh`

Protected endpoints:

- All other endpoints in this document.

## Roles

User roles are:

- `owner`
- `admin`
- `member`

`owner` and `admin` are treated as admin users by the backend.

Admin-only endpoints:

- `GET /users`
- `POST /users`
- `PUT /users/{user_id}`
- `POST /connections/test`
- `POST /connections`
- `PUT /connections/{connection_id}`
- `DELETE /connections/{connection_id}`
- `POST /connections/{connection_id}/access`
- `GET /chat/provider-configs`
- `POST /chat/provider-configs/{provider}/test`
- `PUT /chat/provider-configs/{provider}`

Any authenticated user can call the schema snapshot refresh endpoint in the current implementation.

## Common Error Shape

Errors generally return:

```json
{
  "error": "message"
}
```

Common statuses:

- `400` invalid request, unsupported model/provider, invalid SQL, prompt too large, etc.
- `401` missing/invalid auth or invalid refresh token.
- `403` forbidden or inactive user.
- `404` user/conversation/message not found.
- `409` setup unavailable.
- `500` internal server error. The response body is sanitized to `{"error":"internal server error"}`.

Timestamps are JSON strings. Auth token expiry uses RFC3339 format.

## Auth

### Setup Status

Returns whether the installation still needs an owner/admin account.

```http
GET /api/auth/setup/status
```

Response `200`:

```json
{
  "setup_required": true
}
```

### Setup First User

Creates the initial owner/admin account and returns a session.

```http
POST /api/auth/setup
Content-Type: application/json
```

Body:

```json
{
  "name": "Root User",
  "email": "root@example.com",
  "password": "change-me"
}
```

Response `201`:

```json
{
  "user": {
    "id": 1,
    "email": "root@example.com",
    "name": "Root User",
    "role": "owner",
    "must_change_password": false
  },
  "tokens": {
    "access_token": "jwt",
    "refresh_token": "refresh-token",
    "expires_at": "2026-05-25T12:00:00Z"
  },
  "must_change_password": false
}
```

Notes:

- Available only while setup is allowed by the user service.
- Empty `name`, `email`, or `password` returns `400`.

### Login

```http
POST /api/auth/login
Content-Type: application/json
```

Body:

```json
{
  "email": "root@example.com",
  "password": "change-me"
}
```

Response `200`: same session shape as setup.

### Refresh Session

```http
POST /api/auth/refresh
Content-Type: application/json
```

Body:

```json
{
  "refresh_token": "refresh-token"
}
```

Response `200`: same session shape as setup.

### Logout

```http
POST /api/auth/logout
Authorization: Bearer <access_token>
Content-Type: application/json
```

Body:

```json
{
  "refresh_token": "refresh-token"
}
```

Response `204`: no body.

## Users

User response shape:

```json
{
  "id": 1,
  "email": "root@example.com",
  "name": "Root User",
  "role": "owner",
  "is_active": true,
  "must_change_password": false
}
```

### Get Current User

```http
GET /api/users/me
Authorization: Bearer <access_token>
```

Response `200`: user response.

### List Users

Admin-only.

```http
GET /api/users
Authorization: Bearer <access_token>
```

Response `200`:

```json
[
  {
    "id": 1,
    "email": "root@example.com",
    "name": "Root User",
    "role": "owner",
    "is_active": true,
    "must_change_password": false
  }
]
```

### Create User

Admin-only.

```http
POST /api/users
Authorization: Bearer <access_token>
Content-Type: application/json
```

Body:

```json
{
  "name": "Analyst",
  "email": "analyst@example.com",
  "password": "temporary-password",
  "role": "member"
}
```

Response `201`:

Response `201`: user response.

Notes:

- If `role` is omitted, it defaults to `member`.
- Accepted role strings are `owner`, `admin`, and `member`.

### Update User

Admin-only. Partial update; include only fields that should change.

```http
PUT /api/users/{user_id}
Authorization: Bearer <access_token>
Content-Type: application/json
```

Body:

```json
{
  "name": "Updated Analyst",
  "role": "admin",
  "is_active": true
}
```

Response `200`: user response.

Rules:

- `user_id` must be positive and fit within signed 32-bit integer range.
- Only `name`, `role`, and `is_active` are editable through this endpoint.
- `name`, when provided, cannot be blank.
- `role`, when provided, must be one of `owner`, `admin`, or `member`.

### Change Current User Password

```http
POST /api/users/me/password
Authorization: Bearer <access_token>
Content-Type: application/json
```

Body:

```json
{
  "current_password": "old-password",
  "new_password": "new-password"
}
```

Response `200`: current user response.

## Connections

### List Connections

```http
GET /api/connections
Authorization: Bearer <access_token>
```

Response `200`:

```json
[
  {
    "id": 10,
    "name": "Warehouse",
    "database": "postgres",
    "user_id": 1,
    "is_enabled": true,
    "metadata": {
      "include_namespaces": ["public"],
      "exclude_namespaces": ["information_schema"],
      "include_tables_by_namespace": {
        "public": ["customers", "orders"]
      },
      "exclude_tables_by_namespace": {},
      "include_views": true,
      "include_indexes": true,
      "include_foreign_keys": true,
      "include_comments": false
    }
  }
]
```

Visibility rules:

- Admin and owner users receive all connections.
- Member users receive connections for which they have a connection access grant.
- DSNs are never returned.
- DSNs are encrypted server-side before storage.

### Create Connection

Admin-only.

```http
POST /api/connections
Authorization: Bearer <access_token>
Content-Type: application/json
```

Body:

```json
{
  "name": "Warehouse",
  "database": "postgres",
  "dsn": "postgres://user:pass@host:5432/db?sslmode=require",
  "metadata": {
    "include_namespaces": ["public"],
    "exclude_namespaces": ["information_schema"],
    "include_tables_by_namespace": {
      "public": ["customers", "orders"]
    },
    "exclude_tables_by_namespace": {},
    "include_views": true,
    "include_indexes": true,
    "include_foreign_keys": true,
    "include_comments": false
  }
}
```

Response `201`:

```json
{
  "id": 10,
  "name": "Warehouse",
  "database": "postgres",
  "user_id": 1,
  "is_enabled": true,
  "metadata": {
    "include_namespaces": ["public"],
    "exclude_namespaces": ["information_schema"],
    "include_tables_by_namespace": {
      "public": ["customers", "orders"]
    },
    "exclude_tables_by_namespace": {},
    "include_views": true,
    "include_indexes": true,
    "include_foreign_keys": true,
    "include_comments": false
  }
}
```

Supported `database` values:

- `postgres`
- `mysql`
- `cql`

Notes:

- DSNs are accepted in the request but never returned.
- DSNs are encrypted server-side before storage.
- `metadata` is optional. Omitted boolean fields default to `false`, omitted arrays/maps default to `null` unless the client sends empty arrays/maps.
- Connection responses do not include timestamps.
- Admin users can use every connection without a per-connection access grant.

### Test Connection

Admin-only. Tests an unsaved DSN without creating or updating a connection.

```http
POST /api/connections/test
Authorization: Bearer <access_token>
Content-Type: application/json
```

Body:

```json
{
  "database": "postgres",
  "dsn": "postgres://user:pass@host:5432/db?sslmode=require"
}
```

Response `200`:

```json
{
  "ok": true
}
```

Rules:

- Testable `database` values are `postgres` and `mysql`.
- MySQL accepts either native driver DSNs like `user:pass@tcp(host:3306)/db?parseTime=true` or URL DSNs like `mysql://user:pass@host:3306/db?parseTime=true`.
- `cql` remains a stored connection kind, but this test endpoint returns `400` because there is no CQL driver-backed test implementation.
- Failed connection attempts return `400` with the common error shape.

### Edit Connection

Admin-only. This is a partial update; include only fields that should change.

```http
PUT /api/connections/{connection_id}
Authorization: Bearer <access_token>
Content-Type: application/json
```

Body:

```json
{
  "name": "Warehouse Primary",
  "database": "postgres",
  "dsn": "postgres://user:pass@host:5432/db?sslmode=require",
  "is_enabled": true,
  "metadata": {
    "include_namespaces": ["public", "analytics"],
    "exclude_namespaces": [],
    "include_tables_by_namespace": {},
    "exclude_tables_by_namespace": {},
    "include_views": true,
    "include_indexes": true,
    "include_foreign_keys": true,
    "include_comments": true
  }
}
```

Response `200`: connection response.

Rules:

- `connection_id` must be positive and fit within signed 32-bit integer range.
- `name`, when provided, cannot be blank.
- `database`, when provided, must be one of the supported database values.
- DSN is write-only, encrypted server-side before storage, and is not returned.

### Delete Connection

Admin-only.

```http
DELETE /api/connections/{connection_id}
Authorization: Bearer <access_token>
```

Response `204`: empty body.

### List Connection Access

Admin-only.

```http
GET /api/connections/{connection_id}/access
Authorization: Bearer <access_token>
```

Response `200`:

```json
[
  {
    "user_id": 2,
    "connection_id": 10,
    "can_query": true,
    "allow_writes": false,
    "can_manage": false
  }
]
```

### Grant Connection Access

Admin-only.

```http
POST /api/connections/{connection_id}/access
Authorization: Bearer <access_token>
Content-Type: application/json
```

Body:

```json
{
  "user_id": 2,
  "can_query": true,
  "allow_writes": false,
  "can_manage": false
}
```

Response `201`:

```json
{
  "user_id": 2,
  "connection_id": 10,
  "can_query": true,
  "allow_writes": false,
  "can_manage": false
}
```

## Schema Snapshots

### Refresh Schema Snapshot

Triggers a schema introspection refresh for a connection. This endpoint runs the existing snapshot refresh flow: it introspects the database, stores a new snapshot only if the schema hash changed, and emits a snapshot-created event when a new snapshot is inserted.

```http
POST /api/connections/{connection_id}/schema-snapshot/refresh
Authorization: Bearer <access_token>
```

Response `202`:

```json
{
  "connection_id": 10,
  "status": "accepted"
}
```

Notes for frontend agents:

- The handler accepts authenticated users in the current implementation.
- `connection_id` must be positive and fit within signed 32-bit integer range.
- Even though the response says `accepted`, the backend currently performs the refresh synchronously before returning.
- If embedding is enabled, newly created snapshots are embedded asynchronously via the snapshot-created event subscription.

## Chat Provider Configs

Provider configs are used by the backend to construct LLM clients. API keys are accepted only when saving and are encrypted server-side. Neither plaintext nor encrypted keys are returned.

Known providers:

- `openai`
- `anthropic`
- `gemini`
- `ollama`

### List Provider Configs

Admin-only.

```http
GET /api/chat/provider-configs
Authorization: Bearer <access_token>
```

Response `200`:

```json
[
  {
    "id": 1,
    "provider": "openai",
    "display_name": "OpenAI",
    "base_url": "https://api.openai.com",
    "is_enabled": true,
    "has_api_key": true,
    "metadata": {},
    "created_at": "2026-05-25T12:00:00Z",
    "updated_at": "2026-05-25T12:00:00Z"
  }
]
```

### Save Provider Config

Admin-only. Upserts by provider, so there is at most one config per provider.

```http
PUT /api/chat/provider-configs/{provider}
Authorization: Bearer <access_token>
Content-Type: application/json
```

Body:

```json
{
  "display_name": "OpenAI",
  "api_key": "sk-...",
  "base_url": "https://api.openai.com",
  "is_enabled": true,
  "metadata": {}
}
```

Response `200`:

```json
{
  "id": 1,
  "provider": "openai",
  "display_name": "OpenAI",
  "base_url": "https://api.openai.com",
  "is_enabled": true,
  "has_api_key": true,
  "metadata": {},
  "created_at": "2026-05-25T12:00:00Z",
  "updated_at": "2026-05-25T12:00:00Z"
}
```

Request rules:

- `display_name` is required.
- For `openai`, `anthropic`, and `gemini`, `api_key` is required when creating a new provider config.
- On update, omit `api_key` to preserve the existing stored key.
- `ollama` can be created without `api_key`; use `base_url` for the Ollama server, such as `http://localhost:11434`.
- If `is_enabled` is omitted, it defaults to `true`.
- If `metadata` is omitted, it defaults to `{}`.

### Test Provider Config

Admin-only. Tests an unsaved provider config without creating or updating it. The backend creates a transient provider client and lists models.

```http
POST /api/chat/provider-configs/{provider}/test
Authorization: Bearer <access_token>
Content-Type: application/json
```

Body:

```json
{
  "display_name": "OpenAI",
  "api_key": "sk-...",
  "base_url": "https://api.openai.com",
  "metadata": {}
}
```

Response `200`:

```json
{
  "ok": true,
  "model_count": 2
}
```

Rules:

- `provider` must be one of the known providers.
- `display_name` is required.
- For `openai`, `anthropic`, and `gemini`, `api_key` is required when testing a new provider config.
- When testing an existing provider config, omit `api_key` to reuse the stored encrypted key.
- Provider auth, base URL, or model-listing failures return `400` with the common error shape.

## Models

### List Available Models

```http
GET /api/chat/models
Authorization: Bearer <access_token>
```

Response `200`:

```json
[
  {
    "id": "openai:gpt-5.2",
    "provider": "openai",
    "display_name": "GPT 5.2",
    "description": "Optional description",
    "is_enabled": true,
    "capabilities": {
      "supports_tool_calling": false,
      "supports_structured_output": true,
      "supports_streaming": false,
      "supports_system_instructions": true,
      "supports_vision": false,
      "max_context_tokens": 128000,
      "max_output_tokens": 8192
    }
  }
]
```

Notes:

- Model IDs returned to the frontend are qualified as `{provider}:{provider_model_id}`.
- The backend queries enabled provider configs and provider client model lists at request time.
- Provider credential or client failures surface as `400` model/provider availability errors.

## Chat Conversations

### Create Conversation

```http
POST /api/chat/conversations
Authorization: Bearer <access_token>
Content-Type: application/json
```

Body:

```json
{
  "connection_id": 10,
  "title": "Revenue questions"
}
```

Response `201`:

```json
{
  "id": 100,
  "user_id": 1,
  "connection_id": 10,
  "title": "Revenue questions",
  "created_at": "2026-05-25T12:00:00Z",
  "updated_at": "2026-05-25T12:00:00Z"
}
```

### List Conversations

```http
GET /api/chat/conversations?connection_id=10&limit=20&offset=0
Authorization: Bearer <access_token>
```

Query parameters:

- `connection_id` optional positive integer. Filters to one connection.
- `limit` optional non-negative integer.
- `offset` optional non-negative integer.

Response `200`:

```json
[
  {
    "id": 100,
    "user_id": 1,
    "connection_id": 10,
    "title": "Revenue questions",
    "created_at": "2026-05-25T12:00:00Z",
    "updated_at": "2026-05-25T12:00:00Z"
  }
]
```

### Get Conversation

```http
GET /api/chat/conversations/{conversation_id}
Authorization: Bearer <access_token>
```

Response `200`: single conversation response.

### Delete Conversation

Deletes a conversation owned by the authenticated user. Related messages, retrieval records, executions, and LLM call records are removed by database cascade.

```http
DELETE /api/chat/conversations/{conversation_id}
Authorization: Bearer <access_token>
```

Response `204`: empty body.

### List Messages

```http
GET /api/chat/conversations/{conversation_id}/messages?limit=50&offset=0
Authorization: Bearer <access_token>
```

Query parameters:

- `limit` optional non-negative integer.
- `offset` optional non-negative integer.

Response `200`:

```json
[
  {
    "message": {
      "id": 1000,
      "conversation_id": 100,
      "role": "user",
      "content": "How many users signed up last week?",
      "status": "completed",
      "created_at": "2026-05-25T12:00:00Z"
    }
  },
  {
    "message": {
      "id": 1001,
      "conversation_id": 100,
      "role": "assistant",
      "content": "Counts users who signed up last week.",
      "natural_response": "42 users signed up last week.",
      "provider": "openai",
      "model": "openai:gpt-5.2",
      "status": "completed",
      "created_at": "2026-05-25T12:00:03Z"
    },
    "execution": {
      "message_id": 1001,
      "connection_id": 10,
      "database_kind": "postgres",
      "generated_sql": "SELECT count(*) FROM users;",
      "normalized_sql": "select count(*) from users;",
      "result": {
        "columns": [
          {"name": "count", "data_type": "bigint"}
        ],
        "rows": [
          {"count": 42}
        ],
        "row_count": 1,
        "truncated": false,
        "kind": "scalar"
      },
      "execution_latency_ms": 120,
      "executed_at": "2026-05-25T12:00:03Z"
    },
    "retrieval": {
      "message_id": 1001,
      "snapshot_id": 20,
      "query_text": "How many users signed up last week?",
      "retrieved_at": "2026-05-25T12:00:02Z"
    }
  }
]
```

### Send Message

Sends a user prompt to a conversation, retrieves schema context, asks the selected LLM to generate SQL, validates/runs the SQL, and returns the assistant turn.

```http
POST /api/chat/conversations/{conversation_id}/messages
Authorization: Bearer <access_token>
Content-Type: application/json
```

Body:

```json
{
  "content": "How many users signed up last week?",
  "provider": "openai",
  "model": "openai:gpt-5.2",
  "require_natural_response": true
}
```

Response `200`:

```json
{
  "conversation": {
    "id": 100,
    "user_id": 1,
    "connection_id": 10,
    "title": "Revenue questions",
    "created_at": "2026-05-25T12:00:00Z",
    "updated_at": "2026-05-25T12:00:03Z"
  },
  "user_message": {
    "id": 1000,
    "conversation_id": 100,
    "role": "user",
    "content": "How many users signed up last week?",
    "status": "completed",
    "created_at": "2026-05-25T12:00:00Z"
  },
  "assistant_message": {
    "id": 1001,
    "conversation_id": 100,
    "role": "assistant",
    "content": "Counts users who signed up last week.",
    "natural_response": "42 users signed up last week.",
    "provider": "openai",
    "model": "openai:gpt-5.2",
    "status": "completed",
    "created_at": "2026-05-25T12:00:03Z"
  },
  "execution": {
    "message_id": 1001,
    "connection_id": 10,
    "database_kind": "postgres",
    "generated_sql": "SELECT count(*) FROM users;",
    "normalized_sql": "select count(*) from users;",
    "result": {
      "columns": [{"name": "count", "data_type": "bigint"}],
      "rows": [{"count": 42}],
      "row_count": 1,
      "truncated": false,
      "kind": "scalar"
    },
    "execution_latency_ms": 120,
    "executed_at": "2026-05-25T12:00:03Z"
  },
  "retrieval": {
    "message_id": 1001,
    "snapshot_id": 20,
    "query_text": "How many users signed up last week?",
    "retrieved_at": "2026-05-25T12:00:02Z"
  }
}
```

Request rules:

- `content` is required and cannot be blank.
- `provider` must be one of the known providers.
- `model` should be a model ID returned by `GET /chat/models`.
- `require_natural_response` is optional and defaults to `false`. When `true`, the backend makes an additional LLM call after successful SQL execution and stores the user-facing explanation in `assistant_message.natural_response`.

SQL generation and correction behavior:

- The backend first asks the selected LLM to generate one read-only SQL statement from the prompt and retrieved schema context.
- If generated SQL fails validation or the database returns a query execution error, the backend sends the failed SQL and sanitized database error back to the LLM and asks for corrected SQL.
- Correction is attempted at most twice, for three total SQL attempts.
- Runtime failures are not correction-eligible. These include missing DSN, driver/open errors, read-only transaction begin errors, context timeout/cancellation, row scan/iteration errors, and commit errors.
- If SQL correction is exhausted, the assistant message is saved with `status: "failed"` and `error_message`, all successful SQL-generation LLM calls are retained, and no execution row is returned.
- If `require_natural_response` is `true` and answer generation fails after SQL execution succeeds, the assistant message remains `status: "completed"`, the execution row is still returned, `natural_response` is omitted, and `error_message` contains the sanitized answer-generation error.

Important frontend states:

- `400` with embedded snapshot readiness errors means the user should refresh/wait for schema snapshot embedding.
- `400` with provider/model unavailable means provider config or model selection is not currently usable.
- Assistant messages can return `status: "failed"` with `error_message`.
