CREATE TABLE IF NOT EXISTS schema_versions (
    version TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS holdings (
    id TEXT PRIMARY KEY,
    fund_code TEXT NOT NULL,
    fund_name TEXT NOT NULL,
    shares_micros INTEGER NOT NULL CHECK (shares_micros >= 0),
    cost_nav_micros INTEGER NOT NULL CHECK (cost_nav_micros >= 0),
    current_nav_micros INTEGER NOT NULL DEFAULT 0,
    daily_change_bp INTEGER NOT NULL DEFAULT 0,
    opened_on TEXT NOT NULL,
    closed_at TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS investment_plans (
    id TEXT PRIMARY KEY,
    fund_code TEXT NOT NULL,
    fund_name TEXT NOT NULL,
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    frequency TEXT NOT NULL CHECK (frequency IN ('daily', 'weekly', 'monthly')),
    execution_day INTEGER NOT NULL,
    start_date TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'paused')),
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS plan_executions (
    id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL REFERENCES investment_plans(id),
    scheduled_date TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'skipped')),
    amount_cents INTEGER NOT NULL DEFAULT 0,
    nav_micros INTEGER NOT NULL DEFAULT 0,
    error_code TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (plan_id, scheduled_date)
);
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS user_indices (
    symbol TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    market TEXT NOT NULL,
    added_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS search_history (
    keyword TEXT PRIMARY KEY,
    asset_type TEXT NOT NULL DEFAULT 'fund',
    searched_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS asset_history (
    date TEXT PRIMARY KEY,
    total_market_value_micros INTEGER NOT NULL,
    total_cost_micros INTEGER NOT NULL,
    day_profit_micros INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS intraday_ticks (
    fund_code TEXT NOT NULL,
    record_time TEXT NOT NULL,
    pct_bp INTEGER NOT NULL,
    price_micros INTEGER NOT NULL,
    PRIMARY KEY (fund_code, record_time)
);
CREATE TABLE IF NOT EXISTS fund_daily_performance (
    fund_code TEXT NOT NULL,
    date TEXT NOT NULL,
    nav_micros INTEGER NOT NULL,
    daily_growth_bp INTEGER NOT NULL,
    confirmed_nav_micros INTEGER NOT NULL,
    PRIMARY KEY (fund_code, date)
);
