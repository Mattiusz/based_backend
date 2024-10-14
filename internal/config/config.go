package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL         string
	GRPCPort            string
	MaxPoolConns        int32
	MaxConnLifetimeMins time.Duration
}

// LoadConfig retrieves and loads environment variables into the Config structure.
func LoadConfig() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051" // default port
	}

	maxPoolConns := int32(25) // sensible default
	if val, exists := os.LookupEnv("MAX_POOL_CONNS"); exists {
		if parsedVal, err := strconv.Atoi(val); err == nil {
			maxPoolConns = int32(parsedVal)
		} else {
			return nil, fmt.Errorf("invalid value for MAX_POOL_CONNS: %v", err)
		}
	}

	maxConnLifetime := 2 * time.Minute // default 2 minutes
	if val, exists := os.LookupEnv("MAX_CONN_LIFETIME_MINS"); exists {
		if parsedVal, err := strconv.Atoi(val); err == nil {
			maxConnLifetime = time.Duration(parsedVal) * time.Minute
		} else {
			return nil, fmt.Errorf("invalid value for MAX_CONN_LIFETIME_MINS: %v", err)
		}
	}

	return &Config{
		DatabaseURL:         dbURL,
		GRPCPort:            grpcPort,
		MaxPoolConns:        maxPoolConns,
		MaxConnLifetimeMins: maxConnLifetime,
	}, nil
}
