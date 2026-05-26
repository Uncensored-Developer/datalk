# Service Blueprint

This document captures the service-writing pattern used in `apps/backend/services` so it can be reused in a different Go project.

## Core Idea

A service is split into a small set of layers with explicit responsibilities:

1. `services/<domain>`: top-level assembly for the domain.
2. `services/<domain>/api`: public in-process contract other services depend on.
3. `services/<domain>/internal/<domain>`: business logic and use cases.
4. `services/<domain>/internal/storage`: storage port owned by the domain.
5. `services/<domain>/internal/storage/db`: SQL adapter that implements the storage port.
6. `services/<domain>/pkg/<domain>`: domain types shared outside the service.
7. `services/<domain>/pkg/errors`: domain-level sentinel errors.
8. `services/<domain>/events`: event definitions and publishers when the domain emits events.
9. `services/<domain>/events/handler`: event consumers when the domain reacts to async messages.

The result is not “HTTP handler -> DB code”. It is:

`assembly -> api facade -> internal service -> storage interface -> db adapter`

That separation is the main pattern to copy.

## Folder Shape

Use this as the default starting layout:

```text
services/
  orders/
    orders.go
    api/
      api.go
      client.go
      types.go
      testing/
    internal/
      orders/
        service.go
        create_order.go
        get_order.go
        testing/
      storage/
        storage.go
        testing/
      storage/db/
        db.go
        mappers.go
        db_test.go
    pkg/
      orders/
        order.go
      errors/
        errors.go
    events/
      events.go
      send_order_created.go
      handler/
        handlers.go
        handle_order_created.go
```

Smaller services can omit `events`. More complex services can add subpackages under `internal/<domain>` for supporting concerns.

## Layer Rules

### 1. Top-level assembly package

File: `services/<domain>/<domain>.go`

Responsibilities:

- Own service construction.
- Wire concrete dependencies.
- Return a small struct containing the public pieces of the domain.
- Keep setup concerns here, not in business-logic files.

Typical shape:

```go
type Orders struct {
	API     *api.Api
	Service *orders.Service
	Handler *handler.Handler
}

func New(cfg config.Config, conn *sql.DB, logger *slog.Logger) *Orders {
	orderService := orders.NewService(conn, cfg, logger)
	return &Orders{
		API:     api.New(logger, cfg, orderService),
		Service: orderService,
	}
}
```

Observed in this repo:

- Simple domains like `users` and `connections` expose `API`.
- Richer domains like `schemas` also expose `Service` and async `Handler`.

### 2. `api` package

This is a service boundary, not the HTTP transport layer.

Responsibilities:

- Define the interface other services depend on.
- Translate API-facing params into internal service input structs.
- Keep the contract stable even if internal logic changes.
- Generate mocks for cross-service tests.

Patterns:

- `api.go` defines `type Service interface` and `type Api struct`.
- `types.go` holds input DTOs for the facade.
- `client.go` defines the dependency interface used by other services.

Rule of thumb:

- Put transport-independent, service-facing methods here.
- Do not put SQL, heavy validation, or cross-cutting setup here.

### 3. `internal/<domain>` service package

This is where business logic lives.

Responsibilities:

- Own use-case methods.
- Validate input.
- Orchestrate dependencies.
- Construct domain objects.
- Call storage and other domain dependencies.
- Emit events when needed.

Structure:

- `service.go`: service struct and dependency fields.
- One file per use case: `create_x.go`, `get_x.go`, `refresh_x.go`, `retrieve_x.go`.

Typical service shape:

```go
type Service struct {
	*base.Base

	storage storage.Storage
	clock   Clock
}
```

Method style:

- Use `context.Context` on every public method.
- Accept a dedicated input struct for non-trivial writes.
- Keep `Validate()` close to the input struct.
- Return domain types from `pkg/<domain>`.
- Wrap infrastructure failures with context.

Typical use-case shape:

```go
type CreateOrderParams struct {
	CustomerID int32
	TotalCents int64
}

func (p *CreateOrderParams) Validate() error {
	if p.CustomerID <= 0 {
		return errors.New("customer id is required")
	}
	return nil
}

func (s *Service) CreateOrder(ctx context.Context, params CreateOrderParams) (*orders.Order, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	order := orders.Order{
		CustomerID: params.CustomerID,
		TotalCents: params.TotalCents,
		CreatedAt:  time.Now().UTC(),
	}

	if err := s.storage.UpsertOrder(ctx, &order); err != nil {
		return nil, xerrors.Newf("failed to upsert order: %w", err)
	}

	return &order, nil
}
```

### 4. `internal/storage` package

This package is the domain-owned persistence port.

Responsibilities:

- Define exactly what the service needs from persistence.
- Hide query details from business logic.
- Declare list/filter params near the interface.
- Generate mocks for service tests.

Patterns:

- Use intention-revealing methods like `UpsertUser`, `ListConnections`, `InsertSnapshot`.
- Prefer domain-specific filter structs over leaking raw SQL concerns.
- Keep the interface small and driven by use cases, not by tables.

This package should be stable enough to mock in unit tests and flexible enough for multiple implementations later.

### 5. `internal/storage/db` package

This is the SQL adapter.

Responsibilities:

- Implement the storage interface.
- Translate domain objects to DB models and back.
- Handle transactions, pagination, ordering, and SQL-specific constraints.

Patterns:

- `db.go` contains the implementation.
- `mappers.go` converts DB models <-> domain types.
- The adapter embeds a shared DB helper where useful.
- Query builders stay here, not in service code.

Rules:

- Keep DB models out of `internal/<domain>` business logic.
- Mutate the passed domain object when insert/upsert returns generated fields.
- Return domain objects, never generated DB row types.

### 6. `pkg/<domain>` package

This holds the domain types shared outside the service.

Examples:

- `users.User`
- `connections.Connection`
- `schemas.Snapshot`

Rules:

- Keep these types focused on business meaning.
- Avoid binding them to DB or HTTP annotations unless the whole project intentionally does that.
- Put enums and small value objects here too.

### 7. `pkg/errors`

Use this for domain sentinel errors such as not-found cases.

Rules:

- Keep generic validation errors inline in the use case when they are local and simple.
- Put reusable business errors here.

Typical examples:

- `ErrUserNotFound`
- `ErrConnectionNotFound`

### 8. `events` and `events/handler`

Only add these when the domain has async behavior.

Responsibilities:

- `events`: topic names, payload types, publishing helpers.
- `events/handler`: message decoding and delegation into the service.

Rules:

- Keep event payloads small and explicit.
- Decode in handlers, not in the service.
- Call a normal service method from the handler; do not duplicate business logic in the handler.

## Construction Pattern

The constructor usually follows this order:

1. Create logger/config wrapper via `base.Base`.
2. Build the storage adapter.
3. Build external adapters or dependent service interfaces.
4. Create the internal service.
5. Wrap it with the `api` facade.
6. Optionally create event handlers/subscriptions.

This keeps all concrete wiring in one place.

## Testing Pattern

The dominant testing style is:

1. Unit test `internal/<domain>` use cases directly.
2. Mock `internal/storage` and any external interfaces.
3. Use table-driven tests.
4. Assert returned values and error context.
5. Put DB adapter tests in `internal/storage/db`.

Recommended split:

- `internal/<domain>/*_test.go`: business logic tests with mocks.
- `internal/storage/db/*_test.go`: adapter/integration tests.
- `api/*_test.go`: only if the facade contains meaningful translation logic.

## Naming Conventions

Keep the naming consistent:

- Service package: `internal/<domain>`
- Public types: `pkg/<domain>`
- Constructor: `NewService(...)`
- Facade constructor: `api.New(...)`
- Write commands: `CreateX`, `RefreshX`, `RegisterX`
- Read queries: `GetX`, `ListX`, `RetrieveX`
- Input structs: `NewX`, `CreateXParams`, `ListXFilter`

Prefer one use case per file once the service grows beyond a trivial size.

## Dependency Rules

Copy these constraints into the next project:

- Business logic depends on interfaces, not DB implementations.
- DB adapters depend on generated models / SQL helpers, not the other way around.
- Other services depend on the `api` contract, not `internal/<domain>`.
- Shared domain types live in `pkg/<domain>`.
- Cross-service orchestration belongs in the internal service, not in the DB layer.

## What Is Worth Reusing Exactly

If you want to reproduce this style in another project, keep these pieces almost unchanged:

1. The folder split between `api`, `internal/<domain>`, `internal/storage`, and `pkg/<domain>`.
2. The “one file per use case” pattern inside `internal/<domain>`.
3. Domain-owned storage interfaces with mock generation.
4. A small top-level assembly constructor per service.
5. Event publisher/handler packages as optional add-ons, not default clutter.

## Copyable Build Checklist

When creating a new service in another project:

1. Create `pkg/<domain>` and define the core domain types first.
2. Create `internal/storage/storage.go` with only the persistence methods your use cases need.
3. Implement `internal/storage/db`.
4. Create `internal/<domain>/service.go` and wire dependencies there.
5. Add one file per use case in `internal/<domain>`.
6. Add `api/api.go`, `api/types.go`, and `api/client.go`.
7. Add `pkg/errors/errors.go` for domain sentinel errors.
8. Add `events` only if the domain emits or handles async messages.
9. Generate mocks for every interface the domain owns.
10. Write unit tests for use cases before expanding the transport layer.

## Minimal Scaffold

If starting from scratch, this is the minimum acceptable version:

```text
services/
  billing/
    billing.go
    api/
      api.go
      types.go
      client.go
    internal/
      billing/
        service.go
        create_invoice.go
        get_invoice.go
      storage/
        storage.go
      storage/db/
        db.go
        mappers.go
    pkg/
      billing/
        invoice.go
      errors/
        errors.go
```

Add transport wiring elsewhere in the application. Do not collapse that back into the service package unless the whole project is intentionally much simpler.
