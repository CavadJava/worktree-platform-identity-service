package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	os.Clearenv()
	cfg := Load()
	if cfg.Port != "8095" {
		t.Errorf("expected default port 8095, got %s", cfg.Port)
	}
	if cfg.JWTTTLMinutes != 60*24 {
		t.Errorf("expected default JWT TTL 1440, got %d", cfg.JWTTTLMinutes)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	os.Clearenv()
	os.Setenv("PORT", "9000")
	os.Setenv("JWT_TTL_MINUTES", "30")
	cfg := Load()
	if cfg.Port != "9000" {
		t.Errorf("expected overridden port 9000, got %s", cfg.Port)
	}
	if cfg.JWTTTLMinutes != 30 {
		t.Errorf("expected overridden JWT TTL 30, got %d", cfg.JWTTTLMinutes)
	}
}
