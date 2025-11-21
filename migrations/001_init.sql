CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE ledger_account_type AS ENUM ('asset', 'liability', 'equity', 'income', 'expense');

CREATE TABLE users (
    id UUID PRIMARY KEY,
    phone TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE stocks (
    symbol TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    exchange TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    corporate_action_factor NUMERIC(18,6) NOT NULL DEFAULT 1,
    lot_size INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO stocks (symbol, name, exchange)
VALUES
    ('RELIANCE', 'Reliance Industries', 'NSE'),
    ('TCS', 'Tata Consultancy Services', 'NSE'),
    ('INFY', 'Infosys', 'NSE')
ON CONFLICT (symbol) DO NOTHING;

CREATE TABLE reward_events (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    symbol TEXT NOT NULL REFERENCES stocks(symbol),
    shares NUMERIC(18,6) NOT NULL CHECK (shares > 0),
    granted_price_inr NUMERIC(18,4) NOT NULL,
    brokerage_inr NUMERIC(18,4) NOT NULL DEFAULT 0,
    taxes_inr NUMERIC(18,4) NOT NULL DEFAULT 0,
    total_cash_out_inr NUMERIC(18,4) NOT NULL,
    rewarded_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_key TEXT NOT NULL UNIQUE
);

CREATE INDEX idx_reward_events_user_time ON reward_events (user_id, rewarded_at);

CREATE TABLE ledger_accounts (
    id SERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    type ledger_account_type NOT NULL,
    currency TEXT,
    symbol TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL REFERENCES reward_events(id),
    account_id INT NOT NULL REFERENCES ledger_accounts(id),
    debit_inr NUMERIC(18,4) NOT NULL DEFAULT 0,
    credit_inr NUMERIC(18,4) NOT NULL DEFAULT 0,
    stock_units NUMERIC(18,6) NOT NULL DEFAULT 0,
    memo TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_positions (
    user_id UUID NOT NULL REFERENCES users(id),
    symbol TEXT NOT NULL REFERENCES stocks(symbol),
    net_shares NUMERIC(18,6) NOT NULL DEFAULT 0,
    avg_cost_inr NUMERIC(18,4) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, symbol)
);

CREATE TABLE price_quotes (
    symbol TEXT PRIMARY KEY REFERENCES stocks(symbol),
    price_inr NUMERIC(18,4) NOT NULL,
    source TEXT NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE price_history (
    symbol TEXT NOT NULL REFERENCES stocks(symbol),
    price_inr NUMERIC(18,4) NOT NULL,
    as_of TIMESTAMPTZ NOT NULL,
    source TEXT NOT NULL,
    PRIMARY KEY (symbol, as_of)
);

CREATE INDEX idx_price_history_symbol_time ON price_history (symbol, as_of);

CREATE TABLE daily_holdings (
    user_id UUID NOT NULL REFERENCES users(id),
    date DATE NOT NULL,
    total_value_inr NUMERIC(18,4) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, date)
);

CREATE TABLE adjustments (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    symbol TEXT NOT NULL REFERENCES stocks(symbol),
    shares NUMERIC(18,6) NOT NULL,
    reason TEXT NOT NULL,
    reference_event UUID REFERENCES reward_events(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO ledger_accounts (code, type)
VALUES ('cash', 'asset'),
       ('brokerage_expense', 'expense'),
       ('tax_expense', 'expense')
ON CONFLICT (code) DO NOTHING;
