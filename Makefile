-include .env
SCHEMA=./migrations/
.PHONY: run migrate-up migrate-down

build: ./cmd/api/build.go
	@go build -o bin/build ./cmd/api/build.go

up:
	@docker-compose up -d db
	@docker-compose exec db sh -c 'until pg_isready -U youruser -d yourdb; do sleep 1; done'

migrate-up:
	@echo "Applying migration files"
	@migrate -path $(SCHEMA) -database "${DB_URL}" up
	@echo "Database migrated successfully"

run:
	@echo "starting server ..."
	@go run ./cmd/api/main.go
	@echo "server started"

migrate-down:
	@echo "Applying database rollback"
	@migrate -path $(SCHEMA) -database "${DB_URL}" down 1

force-migrate:
	@if [ -z "$(version)" ]; then \
		echo "⚠️ Warning: No VERSION provided. Defaulting to 1."; \
	fi
	@migrate -path $(SCHEMA) -database "$(DB_URL)" force $(or $(version),1)

dbversion:
	@migrate -path ./migrations -database "$(DB_URL)" version
