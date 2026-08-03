package main

import (
	"os"
	"testing"
)

func TestDemoSeed_EnvConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/recurso_demoseed?sslmode=disable")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "postgres://user:password@localhost:5432/recurso_demoseed?sslmode=disable" {
		t.Errorf("expected custom DATABASE_URL, got %s", dbURL)
	}
}
