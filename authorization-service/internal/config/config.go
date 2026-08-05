package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	JWTSecret string
	Port      string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		JWTSecret: getEnv("JWT_SECRET", ""),
		Port:      getEnv("PORT", "8084"),
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET must be set")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
