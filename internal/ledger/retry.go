package ledger

import (
	"errors"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrLockExhausted is returned when all retry attempts fail due to lock conflicts
var ErrLockExhausted = errors.New("lock exhausted: could not acquire token lock after max retries")

// isLockNotAvailable checks if a pgx error is PostgreSQL error code 55P03 (lock_not_available)
func isLockNotAvailable(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "55P03"
	}
	return false
}

// RetryWithJitter retries op on PostgreSQL lock_not_available (55P03) errors.
// On each retry it backs off between 10-50ms with random jitter.
// Returns ErrLockExhausted after maxAttempts, or the original error for non-lock errors.
func RetryWithJitter(op func() error, maxAttempts int) error {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := op()
		if err == nil {
			return nil
		}
		if !isLockNotAvailable(err) {
			return err
		}
		if attempt == maxAttempts {
			return ErrLockExhausted
		}
		// Random jitter between 10-50ms
		jitter := time.Duration(10+rand.Intn(41)) * time.Millisecond
		time.Sleep(jitter)
	}
	return ErrLockExhausted
}
