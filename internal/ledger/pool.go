package ledger

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ValidatePostgresConfig queries PostgreSQL configuration and logs warnings
// if settings are below recommended values. It does NOT block startup.
func ValidatePostgresConfig(ctx context.Context, pool *pgxpool.Pool, log interface {
	Warn(string, ...interface{})
}) error {
	checks := []struct {
		param       string
		recommended int64
		unit        string
	}{
		{"max_connections", 200, ""},
		{"max_locks_per_transaction", 16384, ""},
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquiring connection for config check: %w", err)
	}
	defer conn.Release()

	for _, check := range checks {
		var val string
		err := conn.QueryRow(ctx, "SHOW "+check.param).Scan(&val)
		if err != nil {
			continue
		}
		var intVal int64
		fmt.Sscanf(val, "%d", &intVal)
		if intVal < check.recommended {
			log.Warn("PostgreSQL config below recommended value",
				"param", check.param,
				"current", intVal,
				"recommended", check.recommended,
			)
		} else {
			// log at info level - just use fmt for now
			_ = intVal
		}
	}
	return nil
}

// PoolOptions configures the pgxpool connection pool
type PoolOptions struct {
	MaxConns         int32
	MinConns         int32
	MaxConnLifetime  time.Duration
	MaxConnIdleTime  time.Duration
	StatementTimeout time.Duration
}

// DefaultPoolOptions returns production-safe defaults
func DefaultPoolOptions() PoolOptions {
	return PoolOptions{
		MaxConns:         50,
		MinConns:         5,
		MaxConnLifetime:  1 * time.Hour,
		MaxConnIdleTime:  10 * time.Minute,
		StatementTimeout: 5 * time.Second,
	}
}

// NewPool creates a configured pgxpool.Pool with the given DSN and options.
func NewPool(dsn string, opts PoolOptions) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing pool config: %w", err)
	}

	config.MaxConns = opts.MaxConns
	config.MinConns = opts.MinConns
	config.MaxConnLifetime = opts.MaxConnLifetime
	config.MaxConnIdleTime = opts.MaxConnIdleTime

	// Set statement_timeout on each new connection
	timeoutMs := opts.StatementTimeout.Milliseconds()
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return nil // pgxpool handles this differently; set via DSN or BeforeAcquire
	}
	// Add statement_timeout to DSN via options
	if timeoutMs > 0 {
		config.ConnConfig.RuntimeParams["statement_timeout"] = fmt.Sprintf("%d", timeoutMs)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}
	return pool, nil
}
