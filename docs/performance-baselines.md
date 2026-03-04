# Performance Baselines

This document records expected performance targets for the Rubix Go Platform v3.0 ledger layer.

## Targets

| Scenario | p50 Target | p99 Target | Notes |
|----------|-----------|-----------|-------|
| 1000 concurrent transactions | < 100ms | < 500ms | Distinct token sets, full transfer flow |
| 5000 token single transfer | — | lock < 200ms | BatchLockTokens with ANY($1) path |
| 10000 token single transfer | — | lock < 500ms | BatchLockTokens with CTE path |
| Validator history sync (100k entries) | — | < 60s total | Fresh validator joining network |

## PostgreSQL Prerequisites

These targets assume PostgreSQL tuned per `docs/postgresql-tuning.md`:
- `max_locks_per_transaction = 16384`
- `work_mem = 64MB`
- `shared_buffers = 4GB`
- `max_connections = 200`

## Baseline Measurement

Run the load tests after implementation to fill in actual measured values:

```
go test -bench=. -benchtime=30s ./tests/load/...
```

| Scenario | Measured p50 | Measured p99 | Date | Go version | PG version |
|----------|-------------|-------------|------|-----------|-----------|
| (TBD after C10-T01/T02/T03) | | | | | |

## Regression Detection

CI will compare new benchmark results against these baselines. A > 20% increase in p99 triggers a performance regression alert.
