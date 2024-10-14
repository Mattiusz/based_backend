
.PHONY: all build run migrate sqlc protoc dev

# Variables
PROTO_DIR=proto
SQLC_CONFIG=sqlc.yaml
MIGRATIONS_DIR=internal/db/migrations

all: build

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down

sqlc:
	sqlc generate

protoc:
	protoc --go_out=. --go-grpc_out=. $(PROTO_DIR)/*.proto

dev: sqlc protoc