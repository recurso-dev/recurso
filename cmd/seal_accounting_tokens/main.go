package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/secretbox"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@127.0.0.1:5432/recurso?sslmode=disable"
	}

	encKey := os.Getenv("GATEWAY_ENCRYPTION_KEY")
	if encKey == "" {
		log.Fatalf("GATEWAY_ENCRYPTION_KEY is required to seal tokens")
	}

	box, err := secretbox.New([]byte(encKey))
	if err != nil {
		log.Fatalf("Failed to initialize secretbox: %v", err)
	}

	conn, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer func() { _ = conn.Close() }()

	ctx := context.Background()
	acctConnRepo := db.NewAccountingConnectionRepository(conn.DB)
	acctConnRepo.SetVault(box)

	// We'll query all accounting_connections directly to ensure we migrate even inactive ones.
	query := `SELECT id, access_token, COALESCE(refresh_token,'') FROM accounting_connections`
	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		log.Fatalf("Failed to query accounting_connections: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var migrated int
	for rows.Next() {
		var id, accessToken, refreshToken string

		err := rows.Scan(&id, &accessToken, &refreshToken)
		if err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}

		isEncAccess := false
		if _, err := box.Open(accessToken); err == nil {
			isEncAccess = true
		}

		isEncRefresh := false
		if _, err := box.Open(refreshToken); err == nil {
			isEncRefresh = true
		}

		needsUpdate := false
		newAccess := accessToken
		newRefresh := refreshToken

		if !isEncAccess && accessToken != "" {
			sealed, err := box.Seal(accessToken)
			if err != nil {
				log.Fatalf("Failed to seal access token %s: %v", id, err)
			}
			newAccess = sealed
			needsUpdate = true
		}

		if !isEncRefresh && refreshToken != "" {
			sealed, err := box.Seal(refreshToken)
			if err != nil {
				log.Fatalf("Failed to seal refresh token %s: %v", id, err)
			}
			newRefresh = sealed
			needsUpdate = true
		}

		if needsUpdate {
			updateQ := `UPDATE accounting_connections SET access_token = $1, refresh_token = $2 WHERE id = $3`
			_, err = conn.ExecContext(ctx, updateQ, newAccess, newRefresh, id)
			if err != nil {
				log.Fatalf("Failed to update tokens for %s: %v", id, err)
			}
			migrated++
			fmt.Printf("Sealed tokens for connection %s\n", id)
		}
	}

	fmt.Printf("✅ Migration complete. Sealed %d legacy plaintext connections.\n", migrated)
}
