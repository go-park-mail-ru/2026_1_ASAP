package config

import (
	"os"
	"testing"
	"time"
)

func TestServerInfo(t *testing.T) {
	cfg := ServerConfig{
		Host: "127.0.0.1",
		Port: "9000",
	}

	if got := cfg.ServerInfo(); got != "127.0.0.1:9000" {
		t.Fatalf("expected 127.0.0.1:9000, got %s", got)
	}
}

func TestLoadConfigFromEnv_Defaults(t *testing.T) {
	t.Setenv("HOST", "localhost")
	t.Setenv("PORT", "8080")
	t.Setenv("TTL", "3600")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.ServerConfig.Host != "localhost" {
		t.Fatalf("expected host localhost, got %s", cfg.ServerConfig.Host)
	}

	if cfg.ServerConfig.Port != "8080" {
		t.Fatalf("expected port 8080, got %s", cfg.ServerConfig.Port)
	}

	if cfg.SessionConfig.SessionTTL != 3600*time.Second {
		t.Fatalf("expected TTL 3600s, got %v", cfg.SessionConfig.SessionTTL)
	}
}

func TestLoadConfigFromEnv_InvalidTTL(t *testing.T) {
	t.Setenv("TTL", "invalid")
	// Ensure host/port don't fail
	os.Unsetenv("HOST")
	os.Unsetenv("PORT")

	_, err := LoadConfigFromEnv()
	if err == nil {
		t.Fatalf("expected error for invalid TTL, got nil")
	}
}
