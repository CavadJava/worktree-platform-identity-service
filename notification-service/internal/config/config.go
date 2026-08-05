package config

import (
	"os"
	"strings"
)

type Config struct {
	Port               string
	CORSAllowedOrigins []string
	LogServiceURL      string
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8083"),
		CORSAllowedOrigins: strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"), ","),
		LogServiceURL:      getEnv("LOG_SERVICE_URL", "http://localhost:8091"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
