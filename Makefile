
.PHONY: all build run migrate sqlc protoc dev

# Variables
DB_URL ?= postgres://user:password@postgres:5432/myservice_db?sslmode=disable
PROTO_DIR=proto
SQLC_CONFIG=sqlc.yaml
MIGRATIONS_DIR=./migrations

all: build

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

migrate-up:
	migrate -path $(MIGRATION_DIR) -database "$(DB_URL)" up

# Run all down migrations
migrate-down:
	migrate -path $(MIGRATION_DIR) -database "$(DB_URL)" down

sqlc:
	sqlc generate

protoc:
	protoc --go_out=. --go-grpc_out=. $(PROTO_DIR)/*.proto

dev: sqlc protoc