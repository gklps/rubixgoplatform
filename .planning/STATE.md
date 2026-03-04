# Project State

**Project:** Rubix Go Platform — DAG & PostgreSQL Core Migration
**Milestone:** v3.0
**Updated:** 2026-03-05

## Current Status

**Active Phase:** Phase 1 (GitHub issues ready — developers can start)
**Last Action:** GitHub project management complete. Architecture pivoted to pure pgx (no GORM). 95 issues across 7 milestones. All issues enriched with developer-ready content. See `.continue-here.md` for full handoff.

## Architecture Decision (2026-03-05)
**Pure pgx, no GORM.** Fresh network start — no existing data to migrate. All 28 tables use explicit DDL in `InitSchema()`. S-series issues (#565–#580) replace the original GORM-based Phase 1 plan.

## Phase Status

| Phase | Status | Notes |
|-------|--------|-------|
| S0 — pgx Schema Foundation | **ready** | Start: #565 (InitSchema), #566, #567 |
| S1+S2 — Wallet Tables | **blocked on S0** | #568–575, parallel after S0 |
| S3 — Model+Storage Layer | **blocked on S1+S2** | #576–578 |
| S4 — Call Sites + Cleanup | **blocked on S3** | #579–580 |
| Phase 3 — AtomicLedgerWriter | **blocked on S-series** | Start #493+#502 in parallel now |
| Phase 4 — State Integrity | **blocked on Phase 3** | C1+C2+C3 streams |
| Phase 5 — Advanced Features | **blocked on Phase 4** | C5+C8+C9 |
| Phase 6 — Load Testing | **blocked on Phase 5** | #549–554 |
| Audit | **ready any time** | Start #555 (CI) + #556 (lint) now |

## Key Decisions

- **PostgreSQL only** — LevelDB removed from all production paths; SQLite may remain for dev/test
- **DAG over blocks** — `TransactionInfo` replaces `block.TokenChainBlock` as canonical tx record
- **gRPC retained** — not removed in this milestone (later task)
- **Explorer UI out of scope** — Task 2.3 DAG Visualiser is a separate repo concern
- **`tokens` table = current-state cache** — `Token.TransactionID` + new `LatestPosition` + `LatestRole` fields; O(1) validation without tokenchain scan (tokenchain used only for genesis/premint)
- **Pessimistic locking (FOR UPDATE NOWAIT)** — chosen over optimistic CAS for multi-token transactions; sorted token ID lock order prevents deadlock; max 3 retries with jitter
- **Atomic write requirement** — all 4 tables (transactions, tokenchain, tokens, requests) in one `BEGIN/COMMIT`; removes `dbWriteSem` global semaphore in Phase 4
- **Token transaction limit** — hard max 10,000 tokens per transaction; soft limit 1,000; `max_locks_per_transaction = 16384` required in PostgreSQL for large transactions
- **No triggers** — FK constraints enforce referential integrity; triggers explicitly rejected

## Important File Locations

- Entry point: `main.go` → `command/command.go:Run` → `command/node.go:runApp`
- Core struct: `core/core.go` (Core, NewCore, SetupCore)
- Token chain storage (current): `core/wallet/token_chain.go` (LevelDB)
- Token chain storage (target): `core/wallet/` (PostgreSQL via GORM)
- Block format (current): `block/block.go` (CBOR TokenChainBlock)
- Verification (current): `core/token_chain_validation.go`
- Transfer flow (current): `core/transfer.go`, `core/quorum_initiator.go`, `core/quorum_recv.go`
- Storage interface: `core/storage/storage.go`, `core/storage/storage_db.go`
- Config: `core/config/config.go`
- Requirements: `.planning/REQUIREMENTS.md`
- Roadmap: `.planning/ROADMAP.md`

## Context Notes

- Codebase map created: `.planning/codebase/` (7 documents, 2026-03-02)
- Requirements source: `/Users/gokul/Downloads/DAG_Changes_270226.md`
- Branch: `development`
- Security concerns documented in `.planning/codebase/CONCERNS.md` — Phase 8 addresses these
- `log.txt` in root shows `AdvisoryURLStorage` table missing error — fix in Phase 8
