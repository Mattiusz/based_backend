package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabasePort         string
	DatabaseHost         string
	DatabaseUser         string
	DatabasePassword     string
	DatabaseName         string
	GRPCPort             string
	MaxPoolConns         int32
	MaxConnLifetimeMins  time.Duration
	MigrationsDir        string
	KeycloakHost         string
	KeycloakPort         string
	KeycloakRealm        string
	KeycloakClientID     string
	KeycloakClientSecret string
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

	maxConnLifetime := 10 * time.Minute // default 10 minutes
	if val, exists := os.LookupEnv("MAX_CONN_LIFETIME_MINS"); exists {
		if parsedVal, err := strconv.Atoi(val); err == nil {
			maxConnLifetime = time.Duration(parsedVal) * time.Minute
		} else {
			return nil, fmt.Errorf("invalid value for MAX_CONN_LIFETIME_MINS: %v", err)
		}
	}

	keycloakHost := os.Getenv("KEYCLOAK_HOST")
	if keycloakHost == "" {
		keycloakHost = "localhost" // default host
	}

	keycloakPort := os.Getenv("KEYCLOAK_PORT")
	if keycloakPort == "" {
		keycloakPort = "8080" // default port
	}

	keycloakRealm := os.Getenv("KEYCLOAK_REALM")
	if keycloakRealm == "" {
		keycloakRealm = "myrealm" // default realm
	}

	keycloakClientID := os.Getenv("KEYCLOAK_CLIENT_ID")
	if keycloakClientID == "" {
		return nil, fmt.Errorf("KEYCLOAK_CLIENT_ID environment variable is required")
	}

	keycloakClientSecret := os.Getenv("KEYCLOAK_CLIENT_SECRET")
	if keycloakClientSecret == "" {
		return nil, fmt.Errorf("KEYCLOAK_CLIENT_SECRET environment variable is required")
	}

	return &Config{
		DatabasePort:         dbPort,
		DatabaseHost:         dbHost,
		DatabaseUser:         dbUser,
		DatabasePassword:     dbPassword,
		DatabaseName:         dbName,
		GRPCPort:             grpcPort,
		MaxPoolConns:         maxPoolConns,
		MaxConnLifetimeMins:  maxConnLifetime,
		MigrationsDir:        migrationsDir,
		KeycloakHost:         keycloakHost,
		KeycloakPort:         keycloakPort,
		KeycloakRealm:        keycloakRealm,
		KeycloakClientID:     keycloakClientID,
		KeycloakClientSecret: keycloakClientSecret,
	}, nil
}
