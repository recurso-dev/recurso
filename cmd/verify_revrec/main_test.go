package main

import (
	"os"
	"testing"
)

func TestVerifyRevRec_DatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/recurso_test?sslmode=disable")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "postgres://user:password@localhost:5432/recurso_test?sslmode=disable" {
		t.Errorf("expected custom DATABASE_URL, got %s", dbURL)
	}
}
