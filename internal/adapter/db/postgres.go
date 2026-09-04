package db

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

// defaultSlowQueryThreshold is used when SLOW_QUERY_THRESHOLD_MS is unset.
const defaultSlowQueryThreshold = 250 * time.Millisecond

// NewConnection establishes a connection pool to Postgres. Statements slower
// than SLOW_QUERY_THRESHOLD_MS (default 250; 0 disables) are logged at WARN
// with the statement text — see slowlog.go.
func NewConnection(dbURL string) (*sql.DB, error) {
	return NewConnectionWithSlowLog(dbURL, slowQueryThresholdFromEnv(), nil)
}

// NewConnectionWithSlowLog is NewConnection with an explicit slow-query
// threshold and logger (nil = slog.Default()).
func NewConnectionWithSlowLog(dbURL string, threshold time.Duration, logger *slog.Logger) (*sql.DB, error) {
	connector, err := newSlowLogConnector(dbURL, threshold, logger)
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(connector)

	pingCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, err
	}

	// Pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func slowQueryThresholdFromEnv() time.Duration {
	v := os.Getenv("SLOW_QUERY_THRESHOLD_MS")
	if v == "" {
		return defaultSlowQueryThreshold
	}
	ms, err := strconv.Atoi(v)
	if err != nil || ms < 0 {
		slog.Warn("invalid SLOW_QUERY_THRESHOLD_MS, using default", "value", v, "default_ms", defaultSlowQueryThreshold.Milliseconds())
		return defaultSlowQueryThreshold
	}
	return time.Duration(ms) * time.Millisecond
}
