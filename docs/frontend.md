# Frontend Development

The web app lives in `apps/web` and uses Vite, React, and Material UI. During local development, run the backend normally and run Vite separately for fast reloads:

```sh
cd apps/backend
make all
make run
```

```sh
cd apps/web
VITE_API_PROXY_TARGET=http://localhost:8007 make dev
```

Vite proxies `/api` requests to the backend. If the backend is running on a different origin, set `VITE_API_PROXY_TARGET` to that backend URL.

## Production-Style Builds

The backend serves the frontend from one Go binary. For a local production-style binary from the repository root, run:

```sh
make single-binary
```

That target:

1. Builds `apps/web` into `apps/web/dist`.
2. Copies the built assets into `apps/backend/servers/echo/staticweb/dist`.
3. Builds `apps/backend/datalk-api`.

The generated frontend files under `apps/web/dist` and `apps/backend/servers/echo/staticweb/dist` are ignored by git. The tracked `placeholder.txt` only keeps the Go embed package compilable before assets are built.

For container releases, use:

```sh
make docker-release
```

The Docker build performs the same flow in build stages: Node builds the frontend, the Go stage copies those assets into the backend embed directory, and the runtime image contains only the compiled `datalk-api` binary.
