package sync

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FindMissingHistory returns the subset of tokenIDs that have no genesis entry
// (position=1) in the tokenchain table.
func FindMissingHistory(ctx context.Context, pool *pgxpool.Pool, tokenIDs []string) ([]string, error) {
	if len(tokenIDs) == 0 {
		return nil, nil
	}

	// Find which tokens have a genesis (position=1) entry
	rows, err := pool.Query(ctx,
		`SELECT DISTINCT token_id
         FROM tokenchain
         WHERE token_id = ANY($1::text[]) AND position = 1`,
		tokenIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	present := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		present[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var missing []string
	for _, id := range tokenIDs {
		if !present[id] {
			missing = append(missing, id)
		}
	}
	return missing, nil
}

// SafeInsertHistory inserts tokenchain entries idempotently.
// Uses ON CONFLICT DO NOTHING so it is safe to call multiple times with the same data.
// After inserting, updates tokens.latest_position if a newer entry was inserted.
func SafeInsertHistory(ctx context.Context, pool *pgxpool.Pool, entries []TokenChainEntry) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, e := range entries {
		_, err := tx.Exec(ctx,
			`INSERT INTO tokenchain (token_id, transaction_id, role, position)
             VALUES ($1, $2, $3, $4)
             ON CONFLICT (token_id, position) DO NOTHING`,
			e.TokenID, e.TransactionID, e.Role, e.Position,
		)
		if err != nil {
			return err
		}
	}

	// Update tokens table where inserted entries are newer than current latest
	// Group by token_id to find the max position per token in this batch
	tokenMaxPos := make(map[string]TokenChainEntry)
	for _, e := range entries {
		if cur, ok := tokenMaxPos[e.TokenID]; !ok || e.Position > cur.Position {
			tokenMaxPos[e.TokenID] = e
		}
	}

	for _, e := range tokenMaxPos {
		_, err := tx.Exec(ctx,
			`UPDATE tokens
             SET transaction_id = $1, latest_position = $2, latest_role = $3
             WHERE token_id = $4
               AND (latest_position IS NULL OR latest_position < $2)`,
			e.TransactionID, e.Position, e.Role, e.TokenID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// TokenChainEntry represents one link in a token's history chain
type TokenChainEntry struct {
	TokenID       string
	TransactionID string
	Role          int16
	Position      int64
}
