# Mutex Audit: core.go and wallet.go

## Summary

This audit documents all mutex usage in `core/core.go` and `core/wallet/wallet.go` to identify
potential deadlock risks when introducing PostgreSQL row-level locking via `FOR UPDATE NOWAIT`.

---

## core/core.go Mutexes

### 1. `lock` — `sync.RWMutex`

**Field declaration (line 104):**
```go
lock sync.RWMutex
```

**What it protects:**
- `c.started` (bool): whether the core node has been started
- `c.sd` (map[string]*ServiceDetials): service details map (registered p2p services)

**Where it is acquired:**
| Call site | File | Method | Operation |
|-----------|------|--------|-----------|
| `c.lock.Lock()` / `Unlock()` | `core.go:540-542` | `GetStartStatus()` | Read `c.started` |
| `c.lock.Lock()` / `Unlock()` | `core.go:546-548` | `SetStartStatus()` | Write `c.started = true` |
| `c.lock.Lock()` / `Unlock()` | `service.go:59-61` | `AddService()` | Calls `initServices()` which modifies `c.sd` |
| `c.lock.Lock()` / `Unlock()` | `service.go:94-96` | `initServices()` | Write `c.sd[sn] = sd` |
| `c.lock.Lock()` / `Unlock()` | `service.go:107-109` | `startService()` | Read `c.sd[sn]` |
| `c.lock.Lock()` / `Unlock()` | `explorer_service.go:23-25` | explorer init | Read/write service-related state |

**Note:** Despite being `sync.RWMutex`, only the exclusive `Lock()` form is used — no `RLock()` calls exist for this field. It functions as a plain `sync.Mutex`.

---

### 2. `ipfsLock` — `sync.RWMutex`

**Field declaration (line 105):**
```go
ipfsLock sync.RWMutex
```

**What it protects:**
- `c.ipfsState` (bool): whether the IPFS daemon is currently running

**Where it is acquired:**
| Call site | File | Method | Operation |
|-----------|------|--------|-----------|
| `c.ipfsLock.RLock()` / `RUnlock()` | `ipfs.go:300-302` | `GetIPFSState()` | Read `c.ipfsState` |
| `c.ipfsLock.Lock()` / `Unlock()` | `ipfs.go:307-309` | `SetIPFSState()` | Write `c.ipfsState` |

**Note:** This mutex is used correctly — RLock for reads, Lock for writes.

---

### 3. `qlock` — `sync.RWMutex`

**Field declaration (line 106):**
```go
qlock sync.RWMutex
```

**What it protects:**
- `c.quorumRequest` (map[string]*ConsensusStatus): in-flight quorum consensus state keyed by request ID
- `c.pd` (map[string]*PledgeDetails): pledge details map keyed by request ID

**Where it is acquired:**
Multiple sites in `quorum_initiator.go` and `pledge_finality_parallel.go`. Key patterns:
- Around reads/writes of `c.quorumRequest[cr.ReqID]` and `c.pd[cr.ReqID]`
- Held briefly while registering a new consensus request
- Held while reading pledge details to send credits

**Critical observation:** `qlock` is acquired and then released (not deferred) before making network I/O calls (e.g., `p.SendJSONRequest`). This is correct — network I/O is NOT performed while holding the lock.

**Note:** Despite being `sync.RWMutex`, `qlock` is always used as `Lock()`/`Unlock()` — never `RLock()`. It functions as a plain `sync.Mutex`.

---

### 4. `rlock` — `sync.Mutex`

**Field declaration (line 107):**
```go
rlock sync.Mutex
```

**What it protects:**
- `c.webReq` (map[string]*did.DIDChan): map of in-flight web/API request state, keyed by request ID

**Where it is acquired:**
| Call site | File | Method | Operation |
|-----------|------|--------|-----------|
| `c.rlock.Lock()` / `Unlock()` | `core.go:715-716` | `AddWebReq()` | Write `c.webReq[req.ID]` |
| `c.rlock.Lock()` / `Unlock()` | `core.go:733-734` | `GetWebReq()` | Read `c.webReq[reqID]` |
| `c.rlock.Lock()` / `Unlock()` | `core.go:743-744` | `UpateWebReq()` | Write `c.webReq[reqID].Req` |
| `c.rlock.Lock()` / `Unlock()` | `core.go:754-755` | `RemoveWebReq()` | Delete from `c.webReq` |

**Note:** `RemoveWebReq()` also acquires `req.PasswordMutex` (a per-entry `sync.RWMutex` inside `did.DIDChan`) while holding `rlock`. See lock ordering note below.

---

### 5. `dbWriteSem` — `chan struct{}` (package-level global semaphore, capacity 1)

**Declaration (line 96):**
```go
var dbWriteSem = make(chan struct{}, 1)
```

**What it serializes:**
- LevelDB/SQLite write operations in `quorum_validation.go` — specifically the critical section that writes token chain state during quorum validation

**Where it is acquired:**
| Call site | File | Operation |
|-----------|------|-----------|
| `dbWriteSem <- struct{}{}` | `quorum_validation.go:334` | Acquire before write |
| `<-dbWriteSem` | `quorum_validation.go:338` | Release after write |
| `dbWriteSem <- struct{}{}` | `quorum_validation.go:349` | Acquire before write |
| `<-dbWriteSem` | `quorum_validation.go:353` | Release after write |

**Note:** This is a capacity-1 channel acting as a mutex/semaphore. It is separate from the Go mutexes above.

---

## core/wallet/wallet.go Mutexes

### 1. `ChainDB.l` — `sync.Mutex` (embedded in `ChainDB` struct, line 66)

**Struct:**
```go
type ChainDB struct {
    *leveldb.DB
    l sync.Mutex
}
```

**What it protects:**
- Serializes concurrent LevelDB write operations (Put/Delete) on the embedded `*leveldb.DB`
- Used for the atomic key-migration operations (old key format -> new key format) in `token_chain.go`

**Where it is acquired:**
| Call site | File | Operation |
|-----------|------|-----------|
| `db.l.Lock()` / `Unlock()` | `token_chain.go:339-344` | Delete old key + Put new key (atomic rename) in `updateNewKey()` |
| `db.l.Lock()` / `Unlock()` | `token_chain.go:367-372` | Same in `updateFullNodeNewKey()` |
| `db.l.Lock()` / `Unlock()` | `token_chain.go:610-612` | LevelDB write during token chain append |
| `db.l.Lock()` / `Unlock()` | `token_chain.go:617-619` | LevelDB write during token chain append |
| `db.l.Lock()` / `Unlock()` | `token_chain.go:622-627` | LevelDB write during token chain append |

**Instances used:** `w.tcs`, `w.ntcs`, `w.smartContractTokenChainStorage`, `w.FTChainStorage`, `w.fullNodeStorage`

---

### 2. `Wallet.l` — `sync.Mutex` (line 75)

**Field declaration:**
```go
l sync.Mutex
```

**What it protects:**
- All reads and writes to the SQLite/GORM `TokenStorage` table (the `tokens` table), including:
  - `CreateTokens`, `GetAllTokens`, `GetAllWholeTokens`, `UpdateToken`, `RemoveTokens`, `LockTokens`, `UnlockTokens`, `ReleaseAllLockedTokens`, `GetTokens`, `GetWholeTokens`, `GetTokenState`, etc.
- Rollback operations on token state in `token_rollback.go`
- LevelDB batch writes in `leveldb_rollback.go`

**Where it is acquired:**
Heavily used in `core/wallet/token.go` (25+ call sites) and `core/wallet/token_rollback.go` (10+ call sites) and `core/wallet/leveldb_rollback.go` (5+ call sites). All follow the pattern:
```go
w.l.Lock()
defer w.l.Unlock()
// ... DB operation ...
```

---

### 3. `Wallet.dtl` — `sync.Mutex` (line 76)

**Field declaration:**
```go
dtl sync.Mutex
```

**What it protects:**
- Write operations to NFT token chain storage (`w.ntcs` LevelDB)
- Smart contract token chain storage (`w.smartContractTokenChainStorage` LevelDB)
- NFT and SmartContract records in the SQLite/GORM layer

**Where it is acquired:**
| Call site | File | Operation |
|-----------|------|-----------|
| `w.dtl.Lock()` | `smart_contract.go:60-61` | Smart contract chain write |
| `w.dtl.Lock()` | `smart_contract.go:77-78` | Smart contract chain write |
| `w.dtl.Lock()` | `smart_contract.go:91-92` | Smart contract chain write |
| `w.dtl.Lock()` | `nft.go:81-82` | NFT chain write |
| `w.dtl.Lock()` | `nft.go:94-95` | NFT chain write |
| `w.dtl.Lock()` | `nft.go:110-111` | NFT chain write |

---

### 4. `Wallet.wl` — `sync.Mutex` (line 78)

**Field declaration:**
```go
wl sync.Mutex
```

**What it protects:**
- No call sites found in `wallet/*.go` via grep. This field appears to be declared but currently unused.

**Status:** Unused / reserved for future use.

---

## Deadlock Risk Analysis

### Risk: Mutex + PostgreSQL Row Lock ordering

PostgreSQL `FOR UPDATE NOWAIT` acquires row locks at the DB level. If a goroutine holds a Go mutex
and then attempts to acquire a PostgreSQL row lock, and another goroutine holds that PostgreSQL row
lock and attempts to acquire the Go mutex, a deadlock can occur.

**Mitigation:** Always acquire PostgreSQL row locks BEFORE acquiring Go mutexes. The `AtomicLedgerWriter`
must not hold any Go mutex when calling `pool.Begin()`.

#### Risk 1: `Wallet.l` wrapping PostgreSQL calls (HIGH)

**Scenario:** `w.l.Lock()` is acquired in `token.go` functions such as `UpdateToken`, `LockTokens`, `UnlockTokens`. If these functions internally use the PostgreSQL storage backend (via GORM) and GORM executes a `FOR UPDATE NOWAIT`, a second goroutine could hold the PostgreSQL row lock and be waiting on `w.l`. This creates a deadlock cycle:

- Goroutine A: holds `w.l`, waiting for PG row lock on token T
- Goroutine B: holds PG row lock on token T, waiting for `w.l`

**Affected files:** `core/wallet/token.go`, `core/wallet/token_rollback.go`

**Required mitigation:** Any function that issues `FOR UPDATE NOWAIT` must NOT be called while `w.l` is held. The `AtomicLedgerWriter` pattern (Phase 4) must acquire PG row locks first (in sorted token order), and only then update in-memory state under `w.l`.

#### Risk 2: `dbWriteSem` serialising write + `FOR UPDATE NOWAIT` inside the critical section (MEDIUM)

**Scenario:** `dbWriteSem` in `quorum_validation.go` serialises multiple write steps. If a PostgreSQL `FOR UPDATE NOWAIT` is added inside the serialised section and another goroutine holds that PG row lock outside the `dbWriteSem` gate, a deadlock occurs.

**Affected files:** `core/quorum_validation.go`

**Required mitigation:** Per the Phase 4 decision, `dbWriteSem` is to be removed entirely. PostgreSQL transaction atomicity replaces it. Until Phase 4 lands, do not add PostgreSQL locking inside `dbWriteSem` critical sections.

#### Risk 3: `qlock` during consensus + `FOR UPDATE NOWAIT` (LOW)

**Scenario:** `c.qlock` guards in-memory consensus maps. If a `FOR UPDATE NOWAIT` call is ever added inside the `qlock`-protected section in `quorum_initiator.go`, and a concurrent goroutine holds the PG row lock and tries to acquire `qlock`, a deadlock results.

**Current status:** No PG operations are performed inside `qlock` critical sections today. Risk arises only if Phase 4 changes touch `quorum_initiator.go`.

**Required mitigation:** Maintain the current pattern — release `qlock` before making any storage calls.

#### Risk 4: `RemoveWebReq` nested lock order — `rlock` → `req.PasswordMutex` (LOW)

**Scenario:** `RemoveWebReq()` acquires `c.rlock` and then acquires `req.PasswordMutex` inside the critical section. If another code path acquires `req.PasswordMutex` first and then tries to obtain `c.rlock`, a deadlock occurs.

**Current status:** `PasswordMutex` is only ever locked within `RemoveWebReq` (while `rlock` is held) and in DID channel processing (which does not touch `rlock`). No inversion found today.

**Required mitigation:** Document lock ordering: `rlock` is always acquired before `PasswordMutex`. Enforce this rule in code review.

#### Risk 5: `ChainDB.l` + LevelDB internal locks (LOW)

**Scenario:** `ChainDB.l` serialises LevelDB key-rename operations (Delete+Put). LevelDB has its own internal locking. The Go mutex is taken before the LevelDB call, meaning LevelDB's internal mutex is acquired while `ChainDB.l` is held. As long as no other goroutine takes LevelDB's lock and then waits for `ChainDB.l`, there is no deadlock.

**Current status:** LevelDB's internal locks are never directly visible to application code. No risk found.

---

## Action Items

The following issues should be tracked as separate GitHub issues:

1. **[ISSUE] Replace `Wallet.l` mutex with fine-grained row-level locking in PostgreSQL path**
   The single `Wallet.l` mutex covering all token table operations is a bottleneck and a deadlock risk when `FOR UPDATE NOWAIT` is introduced. Phase 4 (`AtomicLedgerWriter`) must ensure no Go mutex wraps a `FOR UPDATE NOWAIT` call.

2. **[ISSUE] Remove `dbWriteSem` global channel semaphore (Phase 4)**
   `dbWriteSem` in `quorum_validation.go` is a global serialisation point incompatible with PostgreSQL row-level locking. Phase 4 decision already mandates its removal; track as a prerequisite for Phase 4 completion.

3. **[ISSUE] Audit `Wallet.wl` unused mutex**
   `Wallet.wl` is declared but has zero call sites. Either document its intended use or remove it to avoid confusion during future lock-ordering analysis.

4. **[ISSUE] `lock` and `qlock` declared as `sync.RWMutex` but used as plain `sync.Mutex`**
   Both fields use only the exclusive `Lock()`/`Unlock()` path. Either change the type to `sync.Mutex` (simpler, lower overhead) or add the appropriate `RLock()` read paths to realise the read-concurrency benefit of `RWMutex`.
