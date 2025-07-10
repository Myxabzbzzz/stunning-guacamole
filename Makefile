# Parameters
DB_NAME=restapi_dev
DB_USER=postgres
DB_PASS=your_password
DB_HOST=localhost
DB_PORT=5432
MIGRATIONS_PATH=migrations
DB_URL=postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

.DEFAULT_GOAL := build

.PHONY: build
build:
	go build -v ./cmd/apiserver

.PHONY: test
test:
	go test -v -race -timeout 30s ./...

.PHONY: migrate-up
migrate-up:
	migrate -database "$(DB_URL)" -path $(MIGRATIONS_PATH) up

.PHONY: migrate-down
migrate-down:
	migrate -database "$(DB_URL)" -path $(MIGRATIONS_PATH) down

.PHONY: migrate-force
migrate-force:
	migrate -database "$(DB_URL)" -path $(MIGRATIONS_PATH) force $(version)

.PHONY: migrate-drop
migrate-drop:
	migrate -database "$(DB_URL)" -path $(MIGRATIONS_PATH) drop -f

.PHONY: createdb
createdb:
	createdb -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) $(DB_NAME)

.PHONY: resetdb
resetdb: migrate-drop createdb migrate-up

.PHONY: run
run:
	go run ./cmd/apiserver/main.go