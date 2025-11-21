# Stocky Rewards (Go)

Go service that models Stocky's reward-ledger, portfolio valuation, and hourly price refresh loop. Users can be rewarded fractional Indian equities, fees are hidden in the UI but tracked internally, and the INR value of the portfolio is continuously recomputed for downstream analytics.

## Features

- `POST /reward` writes an immutable reward event with brokerage/tax fees and balanced double-entry ledger rows.
- `GET /today-stocks/{userId}` surfaces all grants created today (UTC).
- `GET /historical-inr/{userId}` streams daily INR valuations sourced from cached price snapshots.
- `GET /stats/{userId}` summarizes totals granted today per symbol and shows the current portfolio value.
- `GET /portfolio/{userId}` (bonus) gives per-symbol holdings, weighted average cost, mark-to-market value, and unrealized P&L.
- Hourly background job fetches mock NSE/BSE prices, stores them, and recomputes `daily_holdings`.

## API quick list

- `POST /reward` — create a reward event (idempotent on `eventId`).
- `GET /today-stocks/{userId}` — all of today’s rewards for that user.
- `GET /historical-inr/{userId}` — per-day INR valuations up to yesterday.
- `GET /stats/{userId}` — today’s totals + current portfolio INR.
- `GET /portfolio/{userId}` — holdings per symbol with mark-to-market (bonus).

## Tech stack

- Go 1.21, chi router, MongoDB driver, shopspring/decimal for precision math.
- **MongoDB Atlas** (free tier available) instead of PostgreSQL for improved scalability.
- Random price fetcher acts as the external market data feed for now.

## Getting started

### Option 1: Using Docker (Recommended)

1. **Install Docker**
   - Download from https://www.docker.com/products/docker-desktop

2. **Start the server with Docker Compose**
   ```bash
   docker-compose up --build
   ```
   The API will be available at **http://localhost:8080**

### Option 2: Local Go Installation

1. **Prerequisites**
   - Go ≥ 1.21 (https://go.dev/dl/)
   - MongoDB Atlas account (https://www.mongodb.com/cloud/atlas)

2. **Set environment variable**
   - Create `.env` file with your MongoDB connection string:
     ```env
     DATABASE_URL=mongodb+srv://username:password@cluster0.xxxxx.mongodb.net/
     PORT=8080
     ```

3. **Fix Go Installation** (if you see `package X is not in std` errors)
   ```cmd
   setx GOROOT C:\Go
   ```
   Then close and reopen your terminal.

4. **Run the server**
   ```bash
   go mod tidy
   go run ./cmd/server
   ```

The API listens on `localhost:8080` by default.

## Background jobs

`internal/jobs/price_sync.go` launches automatically on startup. Every `PRICE_JOB_INTERVAL` (default `1h`) it:

1. Fetches fresh prices for all active stocks via the random fetcher.
2. Writes the quotes to `price_quotes` and `price_history`.
3. Rebuilds `daily_holdings` for the current day (`shares × latest price` for every `(user,symbol)` combo).

If the price service fails, the job logs the error and tries again on the next tick. API responses report stale timestamps so clients can alert when the data is old.

## Database schema

- MongoDB collections: `stocks`, `price_quotes`, `price_history`, `users`, `reward_events`, `ledger_entries`, `user_positions`, `daily_holdings`.
- Double-entry ledger: `ledger_entries` documents track all transactions.
- Positions updated transactionally within reward creation.
- Price cache + history support both instant lookups and time-series analytics.
- `adjustments` collection keeps audit trails for manual interventions (split corrections, delistings, refunds).

## Edge cases & handling

- **Idempotency / replay**: `reward_events.event_key` is unique; the service returns HTTP 409 for duplicates. Pair this with signed webhooks or mTLS to block tampering.
- **Fractional shares & rounding**: all calculations use `decimal` and values are only rounded at storage precision (6 dp for shares, 4 dp for INR).
- **Stock splits/mergers/delistings**: store the multiplier on `stocks.corporate_action_factor`. A scheduled maintenance script updates `user_positions` and inserts compensating ledger entries + `adjustments` row. Delisted symbols are marked `INACTIVE` so the cron job stops fetching new quotes.
- **Price feed downtime**: the quote table retains the last successful fetch. API responses include `priceAsOf` timestamps so clients can detect stale data; reward intake can optionally block if the quote age exceeds a threshold.
- **Adjustments/refunds**: insert a row in `adjustments`, create reversal ledger entries (credit stock inventory, debit cash), and optionally create a correcting reward event if shares are reissued.
- **Scaling**: partition `reward_events`/`ledger_entries` by month, and push them to a warehouse via CDC. Hot-path APIs (`today-stocks`, `stats`) use aggregated tables (`user_positions`, `daily_holdings`) so they stay O(number of symbols) regardless of history length. Horizontal price workers can coordinate with advisory locks if needed.

## Testing ideas

- Unit-test the reward service for fee math and idempotency.
- Integration tests against a Postgres test container for repository methods and HTTP endpoints.
- Property-based tests ensuring debits equal credits per reward event.

## Repository layout

```
cmd/server          Bootstrap + wiring
internal/config     Env parsing
internal/repository Database access (reward, stats, price, ledger)
internal/service    Business use-cases (rewarding, stats, portfolio)
internal/http       REST handlers
internal/price      Mock fetcher + price cache service
internal/jobs       Hourly price sync / valuation job
migrations/         SQL schema
docs/               API + schema documentation
```

## Next steps

- Add authentication/authorization middleware.
- Persist audit logs for adjustment flows.
- Expose health metrics and Prometheus instrumentation for jobs.

Refer to `docs/api.md` for detailed payload examples and to `docs/schema.md` for the data model.
"# Backend" 
