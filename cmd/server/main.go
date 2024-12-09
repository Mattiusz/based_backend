package main

import (
	"context"
	"log"
	"net"

	"github.com/mattiusz/based_backend/internal/config"
	"github.com/mattiusz/based_backend/internal/db"
	"github.com/mattiusz/based_backend/internal/gen/sqlc"
	repository "github.com/mattiusz/based_backend/internal/repositories"
	"google.golang.org/grpc"
)

func main() {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize database
	dbPool, err := db.NewDB(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Run migrations
	if err := db.RunMigrations(ctx, cfg); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Initialize queries and repository
	queries := sqlc.New(dbPool)
	user_repo := repository.NewUserRepository(queries)
	event_repo := repository.NewEventRepository(queries)
	chat_repo := repository.NewChatRepository(queries)

	// Initialize gRPC server
	grpcServer := grpc.NewServer()
	service.RegisterUserServiceServer(grpcServer, service.NewUserService(repo))

	// Listen on the configured port
	listener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	log.Printf("gRPC server listening on port %s", cfg.GRPCPort)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve gRPC server: %v", err)
	}
}
