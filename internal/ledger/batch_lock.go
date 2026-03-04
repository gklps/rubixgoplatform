package ledger

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
)

// BatchLockTokens acquires FOR UPDATE NOWAIT row locks on the given token IDs.
// Token IDs are sorted alphabetically to prevent deadlocks.
// For N <= 5000 tokens, uses ANY($1::text[]).
// For N > 5000 tokens, uses a CTE with unnest to avoid query parameter limits.
// Returns LockedToken slice with current transaction_id and latest_position for each token.
func BatchLockTokens(ctx context.Context, tx pgx.Tx, tokenIDs []string) ([]LockedToken, error) {
	if len(tokenIDs) == 0 {
		return nil, nil
	}

	// Sort to prevent deadlocks (consistent lock acquisition order)
	sorted := make([]string, len(tokenIDs))
	copy(sorted, tokenIDs)
	sort.Strings(sorted)

	var rows pgx.Rows
	var err error

	if len(sorted) <= 5000 {
		// ANY($1::text[]) path — efficient for small to medium batches
		rows, err = tx.Query(ctx,
			`SELECT token_id, transaction_id, latest_position
             FROM tokens
             WHERE token_id = ANY($1::text[])
             ORDER BY token_id
             FOR UPDATE NOWAIT`,
			sorted,
		)
	} else {
		// CTE path — avoids PostgreSQL parameter limits for very large batches
		rows, err = tx.Query(ctx,
			`WITH ids AS (
                SELECT unnest($1::text[]) AS token_id
            )
            SELECT t.token_id, t.transaction_id, t.latest_position
            FROM tokens t
            JOIN ids i ON t.token_id = i.token_id
            ORDER BY t.token_id
            FOR UPDATE NOWAIT`,
			sorted,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("acquiring token locks: %w", err)
	}
	defer rows.Close()

	var locked []LockedToken
	for rows.Next() {
		var lt LockedToken
		var txID *string
		var pos *int64
		if err := rows.Scan(&lt.TokenID, &txID, &pos); err != nil {
			return nil, fmt.Errorf("scanning locked token: %w", err)
		}
		if txID != nil {
			lt.TransactionID = *txID
		}
		if pos != nil {
			lt.LatestPosition = *pos
		}
		locked = append(locked, lt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return locked, nil
}
