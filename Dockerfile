FROM node:22-alpine AS web

WORKDIR /src

COPY apps/web/package.json apps/web/package-lock.json apps/web/
RUN cd apps/web && npm ci

COPY apps/web apps/web
COPY apps/backend/servers/echo/staticweb apps/backend/servers/echo/staticweb
RUN cd apps/web && npm run build

FROM golang:1.25-alpine AS backend

WORKDIR /src

RUN apk add --no-cache build-base

COPY apps/backend/go.mod apps/backend/go.sum apps/backend/
RUN cd apps/backend && go mod download

COPY apps/backend apps/backend
COPY --from=web /src/apps/backend/servers/echo/staticweb/dist apps/backend/servers/echo/staticweb/dist
RUN cd apps/backend && CGO_ENABLED=1 go build -o /out/datalk-api ./cmd/api

FROM alpine:3.22 AS runtime

WORKDIR /app

RUN apk add --no-cache ca-certificates sqlite-libs

COPY --from=backend /out/datalk-api /app/datalk-api

EXPOSE 8007

ENTRYPOINT ["/app/datalk-api"]
