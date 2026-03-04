// Command genesis-init initializes a fresh Rubix network database.
// It creates all required tables, verifies schema, and prints table sizes.
//
// Usage:
//
//	RUBIX_DB_URL=postgres://user:pass@host/db genesis-init
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := os.Getenv("RUBIX_DB_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "error: RUBIX_DB_URL environment variable not set")
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := initSchema(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "error: initializing schema: %v\n", err)
		os.Exit(1)
	}

	if err := verifySchema(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "error: schema verification failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Genesis initialization complete.")
	fmt.Println("Tables: transactions, tokenchain, tokens, requests, state_roots")
}

// initSchema creates all required tables
func initSchema(ctx context.Context, pool *pgxpool.Pool) error {
	ddl := `
CREATE TABLE IF NOT EXISTS transactions (
    id         TEXT PRIMARY KEY,
    info       TEXT,
    signature  TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tokenchain (
    token_id       TEXT NOT NULL,
    transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE RESTRICT,
    role           SMALLINT NOT NULL,
    position       BIGINT NOT NULL,
    CONSTRAINT pk_tokenchain PRIMARY KEY (token_id, position)
);

CREATE INDEX IF NOT EXISTS idx_tokenchain_latest ON tokenchain (token_id, position DESC);

CREATE TABLE IF NOT EXISTS tokens (
    token_id        TEXT PRIMARY KEY,
    transaction_id  TEXT REFERENCES transactions(id) DEFERRABLE INITIALLY DEFERRED,
    latest_position BIGINT,
    latest_role     SMALLINT
);

CREATE TABLE IF NOT EXISTS requests (
    request_id     TEXT PRIMARY KEY,
    transaction_id TEXT,
    status         TEXT NOT NULL,
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS state_roots (
    id           BIGSERIAL PRIMARY KEY,
    block_height BIGINT NOT NULL,
    state_root   TEXT NOT NULL,
    token_count  BIGINT NOT NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_state_roots_height ON state_roots (block_height DESC);
`

	_, err := pool.Exec(ctx, ddl)
	return err
}

// verifySchema checks that all required tables exist via pg_catalog
func verifySchema(ctx context.Context, pool *pgxpool.Pool) error {
	required := []string{"transactions", "tokenchain", "tokens", "requests", "state_roots"}

	for _, table := range required {
		var exists bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS (
                SELECT 1 FROM pg_catalog.pg_tables
                WHERE schemaname = 'public' AND tablename = $1
            )`, table).Scan(&exists)
		if err != nil {
			return fmt.Errorf("checking table %s: %w", table, err)
		}
		if !exists {
			return fmt.Errorf("table %s was not created", table)
		}
	}
	return nil
}
