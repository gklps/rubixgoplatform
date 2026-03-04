# PostgreSQL Tuning Guide for Rubix Go Platform v3.0

## Required Configuration

The following PostgreSQL parameters are required for correct operation, especially for large token transactions.

### max_locks_per_transaction

**Required value:** `16384` (default: 64)

This setting controls how many object locks PostgreSQL can hold per transaction. Each token in a multi-token transfer requires a row-level lock. Without this setting, transactions with > ~3000 tokens will fail with:

```
ERROR: out of shared memory
HINT: You might need to increase max_locks_per_transaction.
```

```sql
-- Check current value
SHOW max_locks_per_transaction;

-- Set permanently (requires restart)
ALTER SYSTEM SET max_locks_per_transaction = 16384;
SELECT pg_reload_conf();  -- or restart PostgreSQL
```

### work_mem

**Recommended value:** `64MB` (default: 4MB)

Controls memory available for sort operations per query. Important for large tokenchain queries.

```sql
SHOW work_mem;
ALTER SYSTEM SET work_mem = '64MB';
SELECT pg_reload_conf();
```

### shared_buffers

**Recommended value:** `4GB` (25% of total RAM)

Controls PostgreSQL's shared memory buffer cache. Larger values reduce disk I/O for hot data.

```sql
SHOW shared_buffers;
ALTER SYSTEM SET shared_buffers = '4GB';
-- Requires PostgreSQL restart
```

### max_connections

**Recommended value:** `200`

The pgxpool is configured with `MaxConns=50`. With multiple node processes, set `max_connections = 200`.

```sql
SHOW max_connections;
ALTER SYSTEM SET max_connections = 200;
-- Requires PostgreSQL restart
```

## Verification

Run this query after applying configuration:

```sql
SELECT name, setting, unit, short_desc
FROM pg_settings
WHERE name IN (
    'max_locks_per_transaction',
    'work_mem',
    'shared_buffers',
    'max_connections'
);
```

## Startup Validation

The Rubix node checks these settings at startup and logs warnings if they are below recommended values:

```
WARN startup PostgreSQL check: max_connections=100 (recommended: >=200)
WARN startup PostgreSQL check: max_locks_per_transaction=64 (recommended: >=16384)
```

## Connection Pool Configuration

The application uses pgxpool with these defaults:
- `MaxConns: 50`
- `MinConns: 5`
- `MaxConnLifetime: 1h`
- `MaxConnIdleTime: 10m`
- `statement_timeout: 5000ms`

## postgresql.conf Example

```ini
max_locks_per_transaction = 16384
work_mem = 64MB
shared_buffers = 4GB
max_connections = 200
```
