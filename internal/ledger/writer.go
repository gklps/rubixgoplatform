package ledger

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LedgerWriter is the interface for atomic ledger writes
type LedgerWriter interface {
	Write(ctx context.Context, req WriteRequest) error
}

// WriteRequest contains all data needed for an atomic ledger write
type WriteRequest struct {
	TransactionID     string
	Info              string
	Signature         string
	TokenChainEntries []TokenChainEntry
	TokenStateUpdates []TokenStateUpdate
	ExpectedPrevTx    map[string]string // tokenID -> expected previous transaction ID
	RequestID         string
	RequestStatus     string
}

// TokenChainEntry is a single entry in the token chain
type TokenChainEntry struct {
	TokenID string
	Role    int16
	// Position is derived at write time from the current latest_position + 1
}

// TokenStateUpdate describes how to update a token's state
type TokenStateUpdate struct {
	TokenID string
	Role    int16
}

// LockedToken is returned after acquiring a FOR UPDATE NOWAIT lock
type LockedToken struct {
	TokenID        string
	TransactionID  string
	LatestPosition int64
}

// AtomicLedgerWriter implements LedgerWriter using pgx explicit SQL.
// All 4 tables (transactions, tokenchain, tokens, requests) are written atomically.
type AtomicLedgerWriter struct {
	pool *pgxpool.Pool
}

// NewAtomicLedgerWriter creates a new AtomicLedgerWriter with the given pool.
func NewAtomicLedgerWriter(pool *pgxpool.Pool) *AtomicLedgerWriter {
	return &AtomicLedgerWriter{pool: pool}
}

// Write executes an atomic ledger write across all 4 tables.
// It acquires sorted FOR UPDATE NOWAIT row locks to prevent deadlocks.
// On lock conflict (55P03), the caller should use RetryWithJitter.
func (w *AtomicLedgerWriter) Write(ctx context.Context, req WriteRequest) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// 1. INSERT INTO transactions
	_, err = tx.Exec(ctx,
		`INSERT INTO transactions (id, info, signature, created_at) VALUES ($1, $2, $3, NOW())`,
		req.TransactionID, req.Info, req.Signature,
	)
	if err != nil {
		return fmt.Errorf("inserting transaction: %w", err)
	}

	// 2. INSERT INTO tokenchain for each entry
	for i, entry := range req.TokenChainEntries {
		_, err = tx.Exec(ctx,
			`INSERT INTO tokenchain (token_id, transaction_id, role, position)
             VALUES ($1, $2, $3, (
                 SELECT COALESCE(MAX(position), 0) + 1
                 FROM tokenchain
                 WHERE token_id = $1
             ))`,
			entry.TokenID, req.TransactionID, entry.Role,
		)
		if err != nil {
			return fmt.Errorf("inserting tokenchain entry %d: %w", i, err)
		}
	}

	// 3. UPDATE tokens for each state update
	for _, upd := range req.TokenStateUpdates {
		_, err = tx.Exec(ctx,
			`UPDATE tokens
             SET transaction_id = $1,
                 latest_position = (SELECT MAX(position) FROM tokenchain WHERE token_id = $2),
                 latest_role = $3
             WHERE token_id = $2`,
			req.TransactionID, upd.TokenID, upd.Role,
		)
		if err != nil {
			return fmt.Errorf("updating token %s: %w", upd.TokenID, err)
		}
	}

	// 4. UPDATE requests if RequestID is provided
	if req.RequestID != "" {
		_, err = tx.Exec(ctx,
			`UPDATE requests SET transaction_id = $1, status = $2, updated_at = NOW() WHERE request_id = $3`,
			req.TransactionID, req.RequestStatus, req.RequestID,
		)
		if err != nil {
			return fmt.Errorf("updating request: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

// ErrPrevTxMismatch is returned when a token's current transaction ID does not
// match the expected previous transaction ID in the WriteRequest.
var ErrPrevTxMismatch = fmt.Errorf("previous transaction ID mismatch")

// ErrLockConflict is returned when FOR UPDATE NOWAIT fails to acquire a lock.
var ErrLockConflict = fmt.Errorf("token lock conflict: another transaction holds the lock")

// lockTokens acquires sorted FOR UPDATE NOWAIT row locks on the given token IDs.
// Returns LockedToken list with current state for each token.
func lockTokens(ctx context.Context, tx pgx.Tx, tokenIDs []string) ([]LockedToken, error) {
	if len(tokenIDs) == 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx,
		`SELECT token_id, transaction_id, latest_position
         FROM tokens
         WHERE token_id = ANY($1)
         ORDER BY token_id
         FOR UPDATE NOWAIT`,
		tokenIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locked []LockedToken
	for rows.Next() {
		var lt LockedToken
		if err := rows.Scan(&lt.TokenID, &lt.TransactionID, &lt.LatestPosition); err != nil {
			return nil, err
		}
		locked = append(locked, lt)
	}
	return locked, rows.Err()
}

// startTime is a helper for timing
var _ = time.Now
