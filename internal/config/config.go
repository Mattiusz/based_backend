package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabasePort              string
	DatabaseHost              string
	DatabaseUser              string
	DatabasePassword          string
	DatabaseName              string
	GRPCPort                  string
	MaxPoolConns              int32
	MaxConnLifetimeMins       time.Duration
	MigrationsDir             string
	AuthentikURL              string
	AuthentikClientID         string
	AuthentikClientSecret     string
	AuthentikRedirectURL      string
	AuthentikTokenEndpoint    string
	AuthentikUserInfoEndpoint string
}

// LoadConfig retrieves and loads environment variables into the Config structure.
func LoadConfig() (*Config, error) {
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432" // default port
	}

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost" // default host
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "user" // default user
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "password" // default password
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "myservice_db" // default database name
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051" // default port
	}

	AuthentikURL := os.Getenv("AUTHENTIK_URL")
	if AuthentikURL == "" {
		return nil, fmt.Errorf("AUTHENTIK_URL must be set")
	}

	AuthentikClientID := os.Getenv("AUTHENTIK_CLIENT_ID")
	if AuthentikClientID == "" {

		return nil, fmt.Errorf("AUTHENTIK_CLIENT_ID must be set")
	}

	AuthentikClientSecret := os.Getenv("AUTHENTIK_CLIENT_SECRET")
	if AuthentikClientSecret == "" {
		return nil, fmt.Errorf("AUTHENTIK_CLIENT_SECRET must be set")
	}

	AuthentikRedirectURL := os.Getenv("AUTHENTIK_REDIRECT_URL")
	if AuthentikRedirectURL == "" {
		return nil, fmt.Errorf("AUTHENTIK_REDIRECT_URL must be set")
	}

	AuthentikTokenEndpoint := os.Getenv("AUTHENTIK_TOKEN_ENDPOINT")
	if AuthentikTokenEndpoint == "" {
		return nil, fmt.Errorf("AUTHENTIK_TOKEN_ENDPOINT must be set")
	}

	AuthentikUserInfoEndpoint := os.Getenv("AUTHENTIK_USER_INFO_ENDPOINT")
	if AuthentikUserInfoEndpoint == "" {
		return nil, fmt.Errorf("AUTHENTIK_USER_INFO_ENDPOINT must be set")
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %v", err)
	}
	rootDir := "file://" + currentDir + "/../.."
	var migrationsDir string
	if os.Getenv("MIGRATIONS_DIR") == "" {
		migrationsDir = rootDir + "/sql/migrations" // default migrations directory
	} else {
		migrationsDir = rootDir + os.Getenv("MIGRATIONS_DIR")
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
		DatabasePort:        dbPort,
		DatabaseHost:        dbHost,
		DatabaseUser:        dbUser,
		DatabasePassword:    dbPassword,
		DatabaseName:        dbName,
		GRPCPort:            grpcPort,
		MaxPoolConns:        maxPoolConns,
		MaxConnLifetimeMins: maxConnLifetime,
		MigrationsDir:       migrationsDir,
	}, nil
}
