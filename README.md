# Mercon

Mercon is a high-throughput Solana data scraper written in Go. It back-fills **all historical transactions** for any wallet address, parses program-specific events (Meteora DLMM & DAMM) plus generic transfers/swaps, and stores everything into structured PostgreSQL databases for later analytics.

---

## Table of Contents

1. Why Mercon?
2. Feature Checklist
3. High-Level Architecture
4. Data-Model Design
5. Worker Lifecycle & Concurrency
6. Error Handling & Observability
7. Configuration
8. Deployment
9. Implementation Roadmap
10. Future Extensions

---

## 1. Why Mercon?

Live-tailing covers new activity, but historical chain data is often missing or incomplete. Mercon continuously drains a Redis queue of wallets and back-fills **every** transaction (from wallet creation up to `NOW()`) so downstream services always have full history.

---

## 2. Feature Checklist

- [ ] Priority queue of wallets (Redis **ZSET**, score ➝ priority)
- [ ] Dynamic, in-process worker pool (auto scales between `MIN_WORKERS` and `MAX_WORKERS`)
- [ ] 50-endpoint RPC pool with per-worker rate-limiting & back-off
- [ ] Accurate, idempotent writes to:
  - Raw chain replica (partitioned PostgreSQL)
  - Parsed analytics DB (DLMM, DAMM, swaps, transfers)
- [ ] Automatic resume from last processed signature on restart
- [ ] Prometheus metrics, zerolog JSON logs, prometheus metrics

---

## 3. High-Level Architecture

```mermaid
graph TD;
  subgraph Redis
    A1[wallet_queue (ZSET)]
    A2[wallet_inflight (HASH)]
    A3[wallet_progress (HASH)]
  end
  subgraph Mercon
    B1[Config Loader]
    B2[RPC Pool]
    B3[Worker Manager]
    B4[Parser Adapter]
    B5[RateLimiter & Retry]
    B6[Metrics & Health]
  end
  subgraph Postgres
    C1[chain schema]
    C2[analytics schema]
  end
  A1 --> B3
  B3 --> B2
  B3 -->|txs| C1
  B4 --> C2
  B3 --> A2
  B3 --> A3
```

---

## 4. Data-Model Design

### 4.1 Raw-Chain Replica (`chain` schema)

| Table                   | Columns (simplified)                                                                           | Notes                   |
| ----------------------- | ---------------------------------------------------------------------------------------------- | ----------------------- |
| `transactions_YYYYMM`   | `id PK`, `slot`, `block_time`, `signature`, `success`, `fee`, `signer_count`, `raw_json JSONB` | Monthly partitions      |
| `instructions_YYYYMM`   | `id PK`, `signature FK`, `idx`, `program_id`, `raw_json JSONB`                                 | 1-to-many               |
| `token_balances_YYYYMM` | `id PK`, `signature FK`, `owner`, `mint`, `pre_amount`, `post_amount`                          | SPL token balance delta |
| `wallet_progress`       | `wallet PK`, `last_sig`, `updated_at`                                                          | Resume checkpoints      |

### 4.2 Parsed / Analytics (`analytics` schema)

Uses existing GORM models (`core/models/*`) plus:

- `swaps`: generic SPL swaps
- `token_transfers`: generic transfers

---

## 5. Worker Lifecycle & Concurrency

1. **Fetch Wallet** – `ZPOPMIN wallet_queue` → wallet with lowest score (highest priority).
2. **Mark In-Flight** – `wallet_inflight[wallet] = worker_id,timestamp`.
3. **Determine Start Sig** – lookup `wallet_progress[wallet]`; default = oldest tx.
4. **Scrape Loop**
   - Pull signatures in batches (e.g., 1 000).
   - Fetch `getTransaction` for each.
   - Write to raw DB ➝ invoke parsers ➝ write processed DB.
   - Update `wallet_progress` every 100 tx.
5. **Complete** – delete from `wallet_inflight`; emit Posthog `wallet_scraped` event.
6. **Crash Recovery** – watchdog re-queues stuck wallets after timeout.

**Dynamic Scaling**

```
desired_workers = clamp(queue_len, MIN_WORKERS, MAX_WORKERS)
```

Workers are added/removed every 30 s. Each holds a **token-bucket** (≈2 req/s) to keep the fleet under free-tier limits (~100 req/s for 50 endpoints).

---

## 6. Error Handling & Observability

- **Rate Limits** – on HTTP 429/503: exponential back-off (250 ms → 30 s). After 5 consecutive 429s an endpoint "cools-down" for 5 min.
- **DB Conflicts** – use `ON CONFLICT DO NOTHING` on unique keys to achieve idempotency.
- **Metrics**
  - `mercon_wallet_queue_length` (gauge)
  - `mercon_workers_active` (gauge)
  - `mercon_rpc_requests_total{status}` (counter)
  - `mercon_wallet_scrape_seconds` (histogram)
- **Logs** – zerolog JSON with `worker_id`, `wallet`, `rpc_endpoint` fields.

---

## 7. Configuration (env vars)

```
REDIS_URL=redis://...
PG_CHAIN_DSN=postgres://...
PG_ANALYTICS_DSN=postgres://...
RPC_ENDPOINTS=url1,url2,...,url50
MIN_WORKERS=4
MAX_WORKERS=50
LOG_LEVEL=info
POSTHOG_KEY=phc_...
```

---

## 8. Deployment

1. **Build** – `make build` produces `build/mercon`.
2. **Systemd Service**

```
[Unit]
Description=Mercon Scraper
After=network.target

[Service]
EnvironmentFile=/opt/mercon/.env
ExecStart=/opt/mercon/mercon
Restart=always

[Install]
WantedBy=multi-user.target
```

3. **Observability** – expose `/metrics` on port 9100; Prometheus scrape; Grafana dashboard JSON stored in repo.

---

## 9. Implementation Roadmap

| Phase                 | Deliverables                                       |
| --------------------- | -------------------------------------------------- |
| 0. Scaffold           | Config loader, logging, metrics HTTP               |
| 1. Queue → Raw DB     | Redis ZSET consumer, wallet checkpoints            |
| 2. Full Raw Replica   | instructions & token_balances tables, partitioning |
| 3. Parser Integration | DLMM / DAMM parsers, swaps & transfers             |
| 4. Robustness         | Retry logic, autoscaling, dashboards               |
| 5. Enhancements       | VIP wallets, multi-instance support                |

---

## 10. Future Extensions

- Live-tail integration to unify historical + real-time pipelines.
- CLI for wallet enqueue / progress inspection.
- Stress-test suite for larger VPS or paid RPC plans.

## 0. Quick Start (TL;DR)

1. Clone repo & bootstrap Go toolchain ≥1.23.
2. Copy `.env.example` → `.env` and fill in **Redis**, **Postgres**, **RPC_ENDPOINTS**.
3. Start local services (Docker compose file pending):
   ```bash
   docker compose up -d redis postgres grafana prometheus
   ```
4. Build & run Mercon:
   ```bash
   cd mercon && make run
   ```
5. Open Grafana at `http://localhost:3000` → dashboard `Mercon Overview`.

> Need a wallet to test? `redis-cli ZADD wallet_queue 0 <WALLET_ADDRESS>` will enqueue it.

---

### Core-Library Integration

Mercon is intentionally _thin_; all heavy-lifting lives in the **`core`** module so other services can share code.

| Responsibility      | Package (module)                        |                    Status                    |
| ------------------- | --------------------------------------- | :------------------------------------------: |
| DB connections      | `core/database`                         |                      ✅                      |
| Raw-chain schema    | `core/database/migrations_raw.go`       |                ⏳ (Phase 1.3)                |
| Analytics schema    | `core/database/migrations_analytics.go` |                 ⏳ (Phase 3)                 |
| GORM models         | `core/models`                           | ✅ (existing) + ⏳ (`Swap`, `TokenTransfer`) |
| Parsers (DLMM/DAMM) | `core/parsers`                          |                      ✅                      |
| Raw insert helpers  | `core/rawchain`                         |                ⏳ (Phase 1.7)                |

Mercon imports these packages—_never_ defines its own models—so there is a single source of truth.

---

### Database Topology

For local development we spin up **one Postgres instance** with **three schemas**:

1. `public` – legacy app tables from `core` (unchanged)
2. `chain` – raw Solana replica (monthly partitions)
3. `analytics` – parsed DLMM/DAMM, swaps, transfers

In production you may split these into distinct databases for IO isolation, but the code assumes schemas by default.

> 🔧 The raw-chain migrations are not in `core` yet (see Phase 1.3). Until then Mercon will skip partition creation and only log a warning.

---
