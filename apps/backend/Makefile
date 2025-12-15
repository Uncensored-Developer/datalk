include .env
export

MIGRATE?=go run -tags "sqlite3 file" github.com/golang-migrate/migrate/v4/cmd/migrate

all: .
	GO111MODULE=on CGO_ENABLED=1 go build -o datalk-api ./cmd/api

run:
	./datalk-api --try-migrate

migrate-new:
	@echo "migration name:"; \
	read name; \
	$(MIGRATE) create -ext sql -dir db/migrations -seq $$name

migrate-up:
	$(MIGRATE) -path=db/migrations/ -database "sqlite3://$${DB_DSN}" up

migrate-up-one:
	$(MIGRATE) -path=db/migrations/ -database "sqlite3://$${DB_DSN}" up 1

migrate-down:
	$(MIGRATE) -path=db/migrations/ -database "sqlite3://$${DB_DSN}" down

migrate-down-one:
	$(MIGRATE) -path=db/migrations/ -database "sqlite3://$${DB_DSN}" down 1

migrate-force:
	@echo "migration version to force:"; \
	read version; \
	$(MIGRATE) -path=db/migrations/ -database "sqlite3://$${DB_DSN}" force $$version

generate-models:
	rm -f db/models/*
	go run github.com/stephenafamo/bob/gen/bobgen-sqlite@latest -c etc/bobgen.yaml

# run gofumpt two times because it's buggy and need 2 pass
	gofumpt -w -l ./db/*/*.go
	gofumpt -w -l ./db/*/*.go
	@echo "generated"

fmt:
# run gofumpt two times because it's buggy and need 2 pass
	@find . -type f -name '*.go' ! -path '*/.cookiecutter/*' ! -path './vendor/*' | xargs gofumpt -l -w
	@find . -type f -name '*.go' ! -path '*/.cookiecutter/*' ! -path './vendor/*' | xargs gofumpt -l -w
