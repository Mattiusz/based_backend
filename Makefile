
.PHONY: all build run migrate sqlc protoc dev

# Variables
DB_URL ?= postgres://user:password@postgres:5432/myservice_db?sslmode=disable
PROTO_DIR=proto
SQLC_CONFIG=sqlc.yaml
MIGRATIONS_DIR=./migrations
GRPC_OUT_DIR=./internal/gen/proto
PROTO_OUT_DIR=./internal/gen/proto

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
	protoc -I proto/ --go_out=$(PROTO_OUT_DIR) --go_opt=paths=source_relative --go-grpc_out=$(GRPC_OUT_DIR) --go-grpc_opt=paths=source_relative $(PROTO_DIR)/*.proto

dev: sqlc protoc

