# Mercon – AI Work Plan

This document enumerates _actionable_ tasks for the AI code-generation agent. Follow these instructions exactly when producing code. Each task is bite-sized and intended to be completed in sequence (or small batches) while continually running `go vet`, `golangci-lint`, and unit tests.

---

## Global Coding Rules

1. **Language**: Go 1.23+ with `go mod` & `go work` setup already in repo.
2. **Style**: `goimports`, `gofumpt`; idiomatic Go, small funcs, descriptive names.
3. **Logging**: Use `zerolog` (already indirect via metfin) with structured JSON.
4. **DB Access**: GORM v1.25; always set context with timeout.
5. **Testing**: `testing` pkg + `stretchr/testify`. Every new package needs tests.
6. **Metrics**: `prometheus/client_golang` – histogram/bucket best-practice.
7. **Config**: All runtime config through env vars; parse once at startup.
8. **Error Handling**: Wrap errors with `%w`; never ignore errors.
9. **Concurrency**: Avoid goroutine leaks; use `errgroup.Group` and contexts.

---

## Phase 0 – Scaffold (Folder: `mercon/internal/app`)

### 0.1 Config Loader

- Create `config/config.go` that reads env vars into a struct, validates, and exposes `Load() (Config, error)`.
- Include slice `[]string RPCEndpoints` and ints `MinWorkers`, `MaxWorkers`.

### 0.2 Logger & Metrics

- `internal/logger/logger.go` – returns a configured `zerolog.Logger`.
- `internal/metrics/metrics.go` – declares Prometheus metrics listed in README.
- `cmd/mercon/main.go` – wire config, logger, metrics HTTP server (`/metrics`, `/ready`).

---

## Phase 1 – Queue ↔ Raw DB Pipeline

### 1.1 Redis Client

- Package `internal/queue` with wrapper around `go-redis` v9.
- Implement methods:
  ```go
  PopWallet(ctx) (string, error)          // ZPOPMIN
  PushWallet(ctx, addr string, priority float64) error
  SetInFlight(ctx, addr, worker string) error
  RemoveInFlight(ctx, addr string) error
  GetProgress(ctx, addr string) (string, error)
  SetProgress(ctx, addr, sig string) error
  ```

### 1.2 Postgres Connection (Raw DB)

- **Use the existing `core/database` package** for connection pooling. Mercon should call `database.InitDB(cfg)` once and reuse `database.GetDB()`. By the way, the raw chain db is not implemented in the core service yet. The core service still uses the old database system with one single database. so you might want to edit that.
- No new DB connection code inside Mercon.

### 1.3 Model Auto-migration

- **Add a new file in `core/database/migrations_raw.go`** that creates monthly partitions dynamically for the `chain` schema. Mercon just invokes `database.RunRawMigrations()`; migration logic resides in core.

### 1.4 Worker Manager

- `internal/worker/manager.go`

  - Accepts Config, Queue, DB, RPC pool.
  - Maintains slice `[]*Worker`.
  - Tick every 30 s → adjust worker count.

- `internal/worker/worker.go`
  - Holds `id`, `rpcClient`, `queue`, `db`.
  - Implements `Start(ctx)`; use `errgroup` inside for batching loop.

### 1.5 RPC Pool

- `internal/rpc/pool.go` – round-robin `[]*http.Client` + endpoint URL.
- Each call wrapped with rate-limiter (`golang.org/x/time/rate`).

### 1.6 Transaction Fetcher

- `internal/rpc/fetch.go` – `FetchTx(ctx, sig string) (*RpcTx, error)` with retries.

### 1.7 Insertion Logic

- Implement helpers in **`core/rawchain/insert.go`** (new package) that insert a transaction + sub-records atomically using the core DB handle; Mercon imports this package. **No GORM models live inside Mercon.**

---

## Phase 2 – Full Raw Replica Extension

1. Add batch insertion for speed (`COPY` or `CreateInBatches`).
2. Benchmark with 10k tx; ensure < 150 µs per row insert avg.
3. Auto-create next month partition 3 days before month-end.

---

## Phase 3 – Parser & Analytics DB

1. Re-use existing code under `core/parsers` via thin adapter.
2. **Add `core/database/analytics.go`** – second GORM connection helper.
3. **Create models inside `core/models`**:
   - `Swap` (generic SPL swaps)
   - `TokenTransfer`
4. Add **`core/database/migrations_analytics.go`** to migrate the new models.
5. Mercon worker uses these core models for inserts; no analytics structs in Mercon.

---

## Phase 4 – Robustness & Autoscaling

1. Implement exponential back-off utility (`internal/backoff/backoff.go`).
2. Cool-down RPC endpoints after repeated 429s.
3. Watchdog goroutine that re-queues stuck wallets: if `wallet_inflight.{wallet}.start_time` older than `X` min, push back to queue.
4. Graceful shutdown: on SIGTERM stop accepting wallets, finish current batch, flush metrics.

---

## Phase 5 – Enhancements

- VIP wallet priority (`queue.PushWallet` with lower score).
- CLI tool (`cmd/merctl`) to enqueue/check wallets.
- Multi-instance advisory lock using `Redlock` if multiple binaries run.

---

## Non-Functional Acceptance Criteria

- Unit test coverage ≥ 70 % for all new packages.
- `go vet` & `golangci-lint run` clean.
- Can scrape **100 wallets** (avg 5 k tx each) on 4-core VPS in < 2 hours.
- Prometheus metrics visible; Grafana dashboard JSON loads without error.

---

## How to Work on This File

**Do not** edit this plan via Finder/Explorer. Use pull-requests with explicit task references (`Phase X.Y`). Tick boxes in README's checklist when tasks merge.
