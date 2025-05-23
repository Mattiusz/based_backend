package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/mattiusz/based_backend/internal/config"
	"github.com/mattiusz/based_backend/internal/db"
	v1 "github.com/mattiusz/based_backend/internal/gen/proto"
	"github.com/mattiusz/based_backend/internal/gen/sqlc"
	"github.com/mattiusz/based_backend/internal/interceptors"
	repository "github.com/mattiusz/based_backend/internal/repositories"
	"github.com/mattiusz/based_backend/internal/services"
)

func grpcPanicRecoveryHandler(p interface{}) error {
	log.Printf("Recovered from panic: %v", p)
	return status.Errorf(codes.Internal, "Internal server error")
}

func main() {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Run migrations
	if err := db.RunMigrations(ctx, cfg); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Initialize database
	dbPool, err := db.NewDB(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Initialize queries and repository
	queries := sqlc.New(dbPool)
	userRepo := repository.NewUserRepository(queries)
	chatRepo := repository.NewChatRepository(queries)
	eventRepo := repository.NewEventRepository(queries)

	// Initialize keycloack client
	authService, err := services.NewAuthService(cfg)
	if err != nil {
		log.Fatalf("failed to create auth service: %v", err)
	}
	authInterceptor := interceptors.NewAuthInterceptor(authService)

	// Initialite grpc services
	userService := services.NewUserService(userRepo)
	chatService := services.NewChatService(chatRepo)
	eventService := services.NewEventService(eventRepo)

	// Initialize gRPC server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			authInterceptor.Unary(),
			recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
		),
		grpc.ChainStreamInterceptor(
			authInterceptor.Stream(),
			recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
		),
	)

	v1.RegisterUserServiceServer(grpcServer, userService)
	v1.RegisterChatServiceServer(grpcServer, chatService)
	v1.RegisterEventServiceServer(grpcServer, eventService)

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
