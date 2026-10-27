package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyDotEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# comment\nPORT=9090\nexport DATABASE_URL=postgres://file\nQUOTED='a b'\nEMPTY=\nBADLINE\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PORT", "7070") // real environment wins over .env
	applyDotEnv(path)
	if os.Getenv("PORT") != "7070" {
		t.Fatalf("environment should take precedence, got %q", os.Getenv("PORT"))
	}
	if os.Getenv("DATABASE_URL") != "postgres://file" {
		t.Fatalf("DATABASE_URL not loaded, got %q", os.Getenv("DATABASE_URL"))
	}
	if os.Getenv("QUOTED") != "a b" {
		t.Fatalf("quotes not stripped, got %q", os.Getenv("QUOTED"))
	}
	if _, exists := os.LookupEnv("EMPTY"); !exists {
		t.Fatal("empty value should still be set")
	}
}

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir) // no .env file here
	for _, k := range []string{"PORT", "DATABASE_URL", "JWT_SECRET", "FRONTEND_ORIGIN", "JWT_TTL_HOURS"} {
		os.Unsetenv(k)
	}
	cfg := Load()
	if cfg.Port != "8080" {
		t.Fatalf("Port = %q", cfg.Port)
	}
	if cfg.JWTTTLHours != 24 {
		t.Fatalf("JWTTTLHours = %d", cfg.JWTTTLHours)
	}
	if cfg.JWTSecret != "" {
		t.Fatalf("JWTSecret should default to empty, got %q", cfg.JWTSecret)
	}
}
