include .env

GOOSE_DRIVER=postgres
GOOSE_DBSTRING=postgresql://$(PG_USER):$(PG_PASSWORD)@$(PG_HOST):$(PG_PORT)/$(PG_NAME)?sslmode=$(PG_SSLMODE)
GOOSE_MIGRATION_DIR=./internal/db/postgres_db/migrations/
GOOSE_ENV=GOOSE_DRIVER="$(GOOSE_DRIVER)" GOOSE_DBSTRING="$(GOOSE_DBSTRING)" GOOSE_MIGRATION_DIR="$(GOOSE_MIGRATION_DIR)"

all: build

run: build
	@./bin/app

build:
	@go mod tidy
	@go build -o ./bin/app ./cmd/api/main.go

clean:
	@rm -rf ./bin

test:
	@go test -v ./tests/...

compose-up:
	@docker-compose up

compose-down:
	@docker-compose down

goose-up:
	@$(GOOSE_ENV) goose up

goose-down:
	@$(GOOSE_ENV) goose down

goose-reset:
	@$(GOOSE_ENV) goose reset

goose-migration:
	@if [ -z "$(name)" ]; then echo "Error: 'name' variable is required."; exit 1; fi
	@$(GOOSE_ENV) goose create -s $(name) sql
