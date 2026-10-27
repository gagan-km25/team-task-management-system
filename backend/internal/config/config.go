package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port, DatabaseURL, JWTSecret, FrontendOrigin string
	JWTTTLHours                                  int
}

// Load reads configuration from the environment, after first applying values
// from an .env file in the working directory (when present). Real environment
// variables win over .env entries so deployments can override file config.
func Load() Config {
	applyDotEnv(".env")
	hours, _ := strconv.Atoi(env("JWT_TTL_HOURS", "24"))
	return Config{
		Port:           env("PORT", "8080"),
		DatabaseURL:    env("DATABASE_URL", "postgres://taskapp:change-this-password@localhost:5432/task_management?sslmode=disable"),
		JWTSecret:      env("JWT_SECRET", ""),
		FrontendOrigin: env("FRONTEND_ORIGIN", "http://localhost:5173"),
		JWTTTLHours:    hours,
	}
}
func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

// applyDotEnv parses a minimal KEY=VALUE file: blank lines and #-comments are
// skipped, an optional "export " prefix is stripped, and matching single or
// double quotes are trimmed from values. Keys already present in the
// environment are left untouched.
func applyDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
			v = v[1 : len(v)-1]
		}
		if k == "" {
			continue
		}
		if _, exists := os.LookupEnv(k); !exists {
			_ = os.Setenv(k, v)
		}
	}
}
