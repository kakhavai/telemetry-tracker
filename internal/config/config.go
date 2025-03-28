package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	ServerPort string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DSN        string // Constructed or provided connection string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	port := getEnv("APP_PORT", "8080")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "") // Should be set via env
	dbName := getEnv("DB_NAME", "telemetry")

	// Basic validation
	if dbPassword == "" {
		slog.Warn("DB_PASSWORD environment variable not set. This is required for database connection.")
		// Depending on policy, you might return an error here:
		// return nil, fmt.Errorf("DB_PASSWORD must be set")
	}
	if _, err := strconv.Atoi(dbPort); err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}
	if _, err := strconv.Atoi(port); err != nil {
		return nil, fmt.Errorf("invalid APP_PORT: %w", err)
	}


	// Prefer DATABASE_URL if provided, otherwise construct DSN
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Construct DSN (consider adding sslmode=require for production with proper CA setup)
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
		slog.Debug("Constructed DSN from individual DB_* variables")
	} else {
		slog.Info("Using DATABASE_URL environment variable for DB connection")
		// Optionally parse DSN here to populate individual fields if needed elsewhere
	}


	return &Config{
		ServerPort: port,
		DBHost:     dbHost,
		DBPort:     dbPort,
		DBUser:     dbUser,
		DBPassword: dbPassword, // Be careful logging this
		DBName:     dbName,
		DSN:        dsn,
	}, nil
}

// Helper function to get environment variables or return default
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}