package main

import (
	"os"
	"testing"
)

func TestSeed_EnvConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/recurso_seed?sslmode=disable")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "postgres://user:password@localhost:5432/recurso_seed?sslmode=disable" {
		t.Errorf("expected custom DATABASE_URL, got %s", dbURL)
	}
}
