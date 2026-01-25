include .env
export

MIGRATE?=go run -tags "postgres file" github.com/golang-migrate/migrate/v4/cmd/migrate

all: .
	GO111MODULE=on CGO_ENABLED=1 go build -o datalk-api ./cmd/api

run:
	./datalk-api --try-migrate

migrate-new:
	@echo "migration name:"; \
	read name; \
	$(MIGRATE) create -ext sql -dir db/migrations -seq $$name

migrate-up:
	$(MIGRATE) -path=db/migrations/ -database "postgres://$${DB_HOSTNAME}:$${DB_PORT}/$${DB_NAME}?user=$${DB_USER}&password=$${DB_PASSWORD}&sslmode=$${DB_SSLMODE}" up

migrate-up-one:
	$(MIGRATE) -path=db/migrations/ -database "postgres://$${DB_HOSTNAME}:$${DB_PORT}/$${DB_NAME}?user=$${DB_USER}&password=$${DB_PASSWORD}&sslmode=$${DB_SSLMODE}" up 1

migrate-down:
	$(MIGRATE) -path=db/migrations/ -database "postgres://$${DB_HOSTNAME}:$${DB_PORT}/$${DB_NAME}?user=$${DB_USER}&password=$${DB_PASSWORD}&sslmode=$${DB_SSLMODE}" down

migrate-down-one:
	$(MIGRATE) -path=db/migrations/ -database "postgres://$${DB_HOSTNAME}:$${DB_PORT}/$${DB_NAME}?user=$${DB_USER}&password=$${DB_PASSWORD}&sslmode=$${DB_SSLMODE}" down 1

migrate-force:
	@echo "migration version to force:"; \
	read version; \
	$(MIGRATE) -path=db/migrations/ -database "postgres://$${DB_HOSTNAME}:$${DB_PORT}/$${DB_NAME}?user=$${DB_USER}&password=$${DB_PASSWORD}&sslmode=$${DB_SSLMODE}" force $$version

generate-models:
	rm -f db/models/*
	go run github.com/stephenafamo/bob/gen/bobgen-psql@latest -c etc/bobgen.yaml

# run gofumpt two times because it's buggy and need 2 pass
	go tool with-modfile gofumpt -w -l ./db/*/*.go
	go tool with-modfile gofumpt -w -l ./db/*/*.go
	@echo "generated"

fmt:
# run gofumpt two times because it's buggy and need 2 pass
	@find . -type f -name '*.go' ! -path '*/.cookiecutter/*' ! -path './vendor/*' | xargs go tool with-modfile gofumpt -l -w
	@find . -type f -name '*.go' ! -path '*/.cookiecutter/*' ! -path './vendor/*' | xargs go tool with-modfile gofumpt -l -w

docker-up:
	docker compose -f docker/docker-compose.yaml -p datalk-backend up
