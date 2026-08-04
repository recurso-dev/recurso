package handler

import (
	"os"
	"testing"

	"github.com/recurso-dev/recurso/internal/validate"
)

// TestMain registers the custom binding validators (currency/country) before
// any handler test runs, mirroring what cmd/api/main.go does at startup — so a
// request struct declaring `binding:"...,currency"` validates in tests too
// instead of panicking on an undefined validation function.
func TestMain(m *testing.M) {
	validate.Register()
	os.Exit(m.Run())
}
