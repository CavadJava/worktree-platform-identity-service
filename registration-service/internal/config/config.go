package config

import (
	"fmt"
	"os"
	"strconv"

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

	Port                string
	NotificationBaseURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	ttl, err := strconv.Atoi(getEnv("JWT_TTL_MINUTES", "60"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_TTL_MINUTES: %w", err)
	}

	cfg := &Config{
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "postgres"),
		DBPassword:          getEnv("DB_PASSWORD", ""),
		DBName:              getEnv("DB_NAME", "postgres"),
		DBSSLMode:           getEnv("DB_SSLMODE", "disable"),
		JWTSecret:           getEnv("JWT_SECRET", ""),
		JWTTTLMinutes:       ttl,
		Port:                getEnv("PORT", "8081"),
		NotificationBaseURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8083"),
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
