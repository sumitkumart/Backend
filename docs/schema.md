# Database Schema

The system targets PostgreSQL (`assignment` database). All money values use `NUMERIC(18,4)` and stock quantities use `NUMERIC(18,6)` so fractional shares and paisa precision are preserved.

## Core entities

| Table | Purpose | Key fields |
| --- | --- | --- |
| `users` | Logical account holder. Users are inserted lazily the first time they earn a reward. | `id UUID PK` |
| `stocks` | Master data for equity symbols; stores the current corporate action multiplier so splits/mergers can be captured. | `symbol PK`, `status`, `corporate_action_factor` |
| `reward_events` | Immutable log of each reward. Ties the user, stock, number of shares, execution price, and invisible fees. `event_key` enforces idempotency. | `event_key UNIQUE`, `references users/stocks` |

## Ledger

| Table | Purpose |
| --- | --- |
| `ledger_accounts` | Chart of accounts. Includes cash, brokerage expense, tax expense, and one stock inventory account per symbol (`stock_inventory:RELIANCE`). |
| `ledger_entries` | Double-entry postings per reward. Stock inventory account debits the acquisition cost and credits cash; brokerage/tax expenses also debit while cash is credited to balance the entry. |

The ledger allows reconciling both rupee outflows and stock units in one stream because entries hold INR debits/credits plus `stock_units`.

## Positions and valuations

| Table | Purpose |
| --- | --- |
| `user_positions` | Running position per `(user_id, symbol)` with weighted-average cost. Updated inside the reward transaction. |
| `price_quotes` | Latest cached INR quote per symbol. Refreshed hourly via the price-sync job. |
| `price_history` | Append-only store of historical price snapshots (each refresh gets written as `symbol + fetched_at`). |
| `daily_holdings` | End-of-day valuations per user. The price job recomputes `shares × latest price` for each user and upserts the value for the current UTC day. `GET /historical-inr` reads from this table. |
| `adjustments` | Manual corrections (refunds, splits, delisting adjustments) with optional linkage to a `reward_event`. Ledger reversal entries accompany every adjustment. |

## Relationships

- `reward_events.user_id → users.id`
- `reward_events.symbol → stocks.symbol`
- `ledger_entries.event_id → reward_events.id`
- `user_positions.user_id → users.id`, `user_positions.symbol → stocks.symbol`
- `price_quotes.symbol`, `price_history.symbol` reference `stocks`
- `daily_holdings.user_id → users.id`

## Corporate actions

When `stocks.corporate_action_factor` changes (e.g., 2:1 split), a maintenance job multiplies `user_positions.net_shares` and divides `avg_cost_inr`. Compensating ledger entries (debit stock inventory, credit corporate action reserve) keep the books honest, and an `adjustments` row captures the reason.

## Indexes

- `reward_events (user_id, rewarded_at)` accelerates lookups for `/today-stocks`.
- `price_history (symbol, as_of)` supports chronological price queries.
- `daily_holdings (user_id, date)` ensures fast `historical-inr` scans.

Refer to `migrations/001_init.sql` for the authoritative SQL.
