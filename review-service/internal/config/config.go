package config

import (
	"os"
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

	AuthorizationBaseURL string

	Port               string
	CORSAllowedOrigins []string
	LogServiceURL      string

	// UploadDir is where review media files land on disk; served back at
	// /media/*. MaxUploadBytes caps a single multipart upload (images and
	// short clips only — this is a demo review system, not a media host).
	UploadDir      string
	MaxUploadBytes int64
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		DBHost:               getEnv("DB_HOST", "localhost"),
		DBPort:               getEnv("DB_PORT", "5433"),
		DBUser:               getEnv("DB_USER", "postgres"),
		DBPassword:           getEnv("DB_PASSWORD", ""),
		DBName:               getEnv("DB_NAME", "postgres"),
		DBSSLMode:            getEnv("DB_SSLMODE", "disable"),
		AuthorizationBaseURL: getEnv("AUTHORIZATION_SERVICE_URL", "http://localhost:8084"),
		Port:                 getEnv("PORT", "8093"),
		CORSAllowedOrigins:   strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"), ","),
		LogServiceURL:        getEnv("LOG_SERVICE_URL", "http://localhost:8091"),
		UploadDir:            getEnv("UPLOAD_DIR", "./uploads"),
		MaxUploadBytes:       25 << 20, // 25MB
	}
}

func (c *Config) DSN() string {
	return "host=" + c.DBHost +
		" port=" + c.DBPort +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" sslmode=" + c.DBSSLMode
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
