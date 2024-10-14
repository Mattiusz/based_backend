package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL string
	GRPCPort    string
}

func LoadConfig() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051" // default port
	}

	return &Config{
		DatabaseURL: dbURL,
		GRPCPort:    grpcPort,
	}, nil
}
