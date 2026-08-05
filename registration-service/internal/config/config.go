package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	JWTSecret     string
	JWTTTLMinutes int

	Port                 string
	NotificationBaseURL  string
	AuthorizationBaseURL string
	CORSAllowedOrigins   []string
	LogServiceURL        string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	ttl, err := strconv.Atoi(getEnv("JWT_TTL_MINUTES", "60"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_TTL_MINUTES: %w", err)
	}

	cfg := &Config{
		DBHost:               getEnv("DB_HOST", "localhost"),
		DBPort:               getEnv("DB_PORT", "5432"),
		DBUser:               getEnv("DB_USER", "postgres"),
		DBPassword:           getEnv("DB_PASSWORD", ""),
		DBName:               getEnv("DB_NAME", "postgres"),
		DBSSLMode:            getEnv("DB_SSLMODE", "disable"),
		JWTSecret:            getEnv("JWT_SECRET", ""),
		JWTTTLMinutes:        ttl,
		Port:                 getEnv("PORT", "8081"),
		NotificationBaseURL:  getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8083"),
		AuthorizationBaseURL: getEnv("AUTHORIZATION_SERVICE_URL", "http://localhost:8084"),
		CORSAllowedOrigins:   strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"), ","),
		LogServiceURL:        getEnv("LOG_SERVICE_URL", "http://localhost:8091"),
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET must be set")
	}

	return cfg, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
