package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	os.Clearenv()
	os.Setenv("JWT_SECRET", "test-secret")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8095" {
		t.Errorf("expected default port 8095, got %s", cfg.Port)
	}
	if cfg.JWTTTLMinutes != 60*24 {
		t.Errorf("expected default JWT TTL 1440, got %d", cfg.JWTTTLMinutes)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	os.Clearenv()
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("PORT", "9000")
	os.Setenv("JWT_TTL_MINUTES", "30")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "9000" {
		t.Errorf("expected overridden port 9000, got %s", cfg.Port)
	}
	if cfg.JWTTTLMinutes != 30 {
		t.Errorf("expected overridden JWT TTL 30, got %d", cfg.JWTTTLMinutes)
	}
}

func TestLoad_MissingJWTSecretFails(t *testing.T) {
	os.Clearenv()
	cfg, err := Load()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET is not set, got nil")
	}
	if cfg != nil {
		t.Errorf("expected nil config on error, got %+v", cfg)
	}
}
