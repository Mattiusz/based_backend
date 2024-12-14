package main

import (
	"context"
	"log"
	"net"

	"github.com/mattiusz/based_backend/internal/config"
	"github.com/mattiusz/based_backend/internal/db"
	v1 "github.com/mattiusz/based_backend/internal/gen/proto"
	"github.com/mattiusz/based_backend/internal/gen/sqlc"
	repository "github.com/mattiusz/based_backend/internal/repositories"
	"github.com/mattiusz/based_backend/internal/services"
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
	userRepo := repository.NewUserRepository(queries)
	chatRepo := repository.NewChatRepository(queries)
	eventRepo := repository.NewEventRepository(queries)

	// Initialite services
	userService := services.NewUserService(userRepo)
	chatService := services.NewChatService(chatRepo)

	// Initialize gRPC server
	grpcServer := grpc.NewServer()
	v1.RegisterUserServiceServer(grpcServer, userService)
	v1.RegisterChatServiceServer(grpcServer, chatService)

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
