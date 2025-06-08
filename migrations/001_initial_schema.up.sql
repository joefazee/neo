
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE countries
(
    id              UUID PRIMARY KEY         DEFAULT uuid_generate_v4(),
    name            VARCHAR(100) NOT NULL UNIQUE,
    code            VARCHAR(3)   NOT NULL UNIQUE,          -- ISO 3166-1 alpha-3
    currency_code   VARCHAR(3)   NOT NULL,                 -- ISO 4217
    currency_symbol VARCHAR(10)  NOT NULL,
    is_active       BOOLEAN                  DEFAULT true,
    config          JSONB                    DEFAULT '{}', -- Country-specific configurations
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Categories table (country-specific)
CREATE TABLE categories
(
    id          UUID PRIMARY KEY         DEFAULT uuid_generate_v4(),
    country_id  UUID         NOT NULL REFERENCES countries (id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(100) NOT NULL,
    description TEXT,
    is_active   BOOLEAN                  DEFAULT true,
    sort_order  INTEGER                  DEFAULT 0,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (country_id, slug)
);

-- Users table with security and KYC
CREATE TABLE users
(
    id                    UUID PRIMARY KEY         DEFAULT uuid_generate_v4(),
    country_id            UUID         NOT NULL REFERENCES countries (id),
    email                 VARCHAR(255) NOT NULL UNIQUE,
    email_verified_at     TIMESTAMP WITH TIME ZONE,
    password_hash         VARCHAR(255) NOT NULL,
    first_name            VARCHAR(100),
    last_name             VARCHAR(100),
    phone                 VARCHAR(20),
    phone_verified_at     TIMESTAMP WITH TIME ZONE,
    date_of_birth         DATE,
    kyc_status            VARCHAR(20)   DEFAULT 'pending' CHECK (kyc_status IN ('pending', 'in_progress', 'verified', 'rejected')),
    kyc_provider          VARCHAR(50),
    kyc_reference         VARCHAR(100),
    kyc_verified_at       TIMESTAMP WITH TIME ZONE,
    two_factor_enabled    BOOLEAN                  DEFAULT false,
    two_factor_secret     VARCHAR(255),
    last_login_at         TIMESTAMP WITH TIME ZONE,
    last_login_ip         INET,
    failed_login_attempts INTEGER                  DEFAULT 0,
    locked_until          TIMESTAMP WITH TIME ZONE,
    is_active             BOOLEAN                  DEFAULT true,
    metadata              JSONB                    DEFAULT '{}',
    created_at            TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at            TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Markets table
CREATE TABLE markets
(
    id                    UUID PRIMARY KEY                  DEFAULT uuid_generate_v4(),
    country_id            UUID                     NOT NULL REFERENCES countries (id),
    category_id           UUID                     NOT NULL REFERENCES categories (id),
    creator_id            UUID REFERENCES users (id),
    title                 VARCHAR(255)             NOT NULL,
    description           TEXT                     NOT NULL,
    market_type           VARCHAR(20)                       DEFAULT 'binary' CHECK (market_type IN ('binary', 'multi_outcome')),
    status                VARCHAR(20)                       DEFAULT 'draft' CHECK (status IN ('draft', 'open', 'closed', 'resolved', 'voided')),
    close_time            TIMESTAMP WITH TIME ZONE NOT NULL,
    resolution_deadline   TIMESTAMP WITH TIME ZONE NOT NULL,
    resolved_at           TIMESTAMP WITH TIME ZONE,
    resolved_outcome      VARCHAR(100),
    resolution_source     TEXT,
    min_bet_amount        DECIMAL(20, 2)           NOT NULL DEFAULT 100.00,
    max_bet_amount        DECIMAL(20, 2),
    total_pool_amount     DECIMAL(20, 2)                    DEFAULT 0.00,
    rake_percentage       DECIMAL(5, 4)                     DEFAULT 0.0500, -- 5%
    creator_revenue_share DECIMAL(5, 4)                     DEFAULT 0.5000, -- 50% of rake
    safeguard_config      JSONB                             DEFAULT '{}',
    oracle_config         JSONB                             DEFAULT '{}',
    metadata              JSONB                             DEFAULT '{}',
    created_at            TIMESTAMP WITH TIME ZONE          DEFAULT NOW(),
    updated_at            TIMESTAMP WITH TIME ZONE          DEFAULT NOW(),
    CONSTRAINT valid_close_time CHECK (close_time > created_at),
    CONSTRAINT valid_resolution_deadline CHECK (resolution_deadline > close_time)
);

-- Market outcomes table
CREATE TABLE market_outcomes
(
    id                 UUID PRIMARY KEY         DEFAULT uuid_generate_v4(),
    market_id          UUID         NOT NULL REFERENCES markets (id) ON DELETE CASCADE,
    outcome_key        VARCHAR(50)  NOT NULL, -- 'yes', 'no', 'openai', 'anthropic', etc.
    outcome_label      VARCHAR(100) NOT NULL,
    sort_order         INTEGER                  DEFAULT 0,
    pool_amount        DECIMAL(20, 2)           DEFAULT 0.00,
    is_winning_outcome BOOLEAN,
    created_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (market_id, outcome_key)
);

-- User wallets
CREATE TABLE wallets
(
    id             UUID PRIMARY KEY         DEFAULT uuid_generate_v4(),
    user_id        UUID       NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    currency_code  VARCHAR(3) NOT NULL,
    balance        DECIMAL(20, 2)           DEFAULT 0.00 CHECK (balance >= 0),
    locked_balance DECIMAL(20, 2)           DEFAULT 0.00 CHECK (locked_balance >= 0),
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (user_id, currency_code)
);

-- Transaction ledger (immutable)
CREATE TABLE transactions
(
    id               UUID PRIMARY KEY         DEFAULT uuid_generate_v4(),
    user_id          UUID           NOT NULL REFERENCES users (id),
    wallet_id        UUID           NOT NULL REFERENCES wallets (id),
    transaction_type VARCHAR(20)    NOT NULL CHECK (transaction_type IN
                                                    ('deposit', 'withdrawal', 'bet_place', 'bet_refund', 'payout',
                                                     'fee')),
    amount           DECIMAL(20, 2) NOT NULL,
    balance_before   DECIMAL(20, 2) NOT NULL,
    balance_after    DECIMAL(20, 2) NOT NULL,
    reference_type   VARCHAR(20), -- 'bet', 'settlement', 'payment'
    reference_id     UUID,
    description      TEXT,
    metadata         JSONB                    DEFAULT '{}',
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Bets table
CREATE TABLE bets
(
    id                 UUID PRIMARY KEY         DEFAULT uuid_generate_v4(),
    user_id            UUID           NOT NULL REFERENCES users (id),
    market_id          UUID           NOT NULL REFERENCES markets (id),
    market_outcome_id  UUID           NOT NULL REFERENCES market_outcomes (id),
    amount             DECIMAL(20, 2) NOT NULL CHECK (amount > 0),
    contracts_bought   DECIMAL(20, 8) NOT NULL CHECK (contracts_bought > 0),
    price_per_contract DECIMAL(20, 2) NOT NULL CHECK (price_per_contract > 0),
    total_cost         DECIMAL(20, 2) NOT NULL CHECK (total_cost > 0),
    transaction_id     UUID           NOT NULL REFERENCES transactions (id),
    status             VARCHAR(20)              DEFAULT 'active' CHECK (status IN ('active', 'settled', 'refunded')),
    settled_at         TIMESTAMP WITH TIME ZONE,
    settlement_amount  DECIMAL(20, 2),
    metadata           JSONB                    DEFAULT '{}',
    created_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Settlement records (immutable)
CREATE TABLE settlements
(
    id              UUID PRIMARY KEY         DEFAULT uuid_generate_v4(),
    market_id       UUID           NOT NULL REFERENCES markets (id),
    user_id         UUID           NOT NULL REFERENCES users (id),
    bet_id          UUID           NOT NULL REFERENCES bets (id),
    settlement_type VARCHAR(20)    NOT NULL CHECK (settlement_type IN ('win', 'loss', 'refund')),
    original_amount DECIMAL(20, 2) NOT NULL,
    payout_amount   DECIMAL(20, 2) NOT NULL  DEFAULT 0.00,
    rake_amount     DECIMAL(20, 2) NOT NULL  DEFAULT 0.00,
    transaction_id  UUID REFERENCES transactions (id),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Payment transactions
CREATE TABLE payment_transactions
(
    id                 UUID PRIMARY KEY         DEFAULT uuid_generate_v4(),
    user_id            UUID           NOT NULL REFERENCES users (id),
    transaction_id     UUID REFERENCES transactions (id),
    provider           VARCHAR(20)    NOT NULL, -- 'paystack', 'flutterwave', 'monnify'
    provider_reference VARCHAR(100)   NOT NULL,
    payment_type       VARCHAR(20)    NOT NULL CHECK (payment_type IN ('deposit', 'withdrawal')),
    amount             DECIMAL(20, 2) NOT NULL,
    currency_code      VARCHAR(3)     NOT NULL,
    status             VARCHAR(20)              DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'success', 'failed', 'cancelled')),
    provider_response  JSONB                    DEFAULT '{}',
    webhook_verified   BOOLEAN                  DEFAULT false,
    created_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Audit logs for security
CREATE TABLE audit_logs
(
    id            UUID PRIMARY KEY         DEFAULT uuid_generate_v4(),
    user_id       UUID REFERENCES users (id),
    action        VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id   UUID,
    old_values    JSONB,
    new_values    JSONB,
    ip_address    INET,
    user_agent    TEXT,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- JWT token blacklist
CREATE TABLE token_blacklist
(
    id         UUID PRIMARY KEY         DEFAULT uuid_generate_v4(),
    token_jti  VARCHAR(255)             NOT NULL UNIQUE,
    user_id    UUID                     NOT NULL REFERENCES users (id),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_country ON users (country_id);
CREATE INDEX idx_users_kyc_status ON users (kyc_status);
CREATE INDEX idx_categories_country_slug ON categories (country_id, slug);
CREATE INDEX idx_markets_status ON markets (status);
CREATE INDEX idx_markets_country_category ON markets (country_id, category_id);
CREATE INDEX idx_markets_close_time ON markets (close_time);
CREATE INDEX idx_market_outcomes_market ON market_outcomes (market_id);
CREATE INDEX idx_wallets_user_currency ON wallets (user_id, currency_code);
CREATE INDEX idx_transactions_user ON transactions (user_id);
CREATE INDEX idx_transactions_created_at ON transactions (created_at);
CREATE INDEX idx_bets_user ON bets (user_id);
CREATE INDEX idx_bets_market ON bets (market_id);
CREATE INDEX idx_bets_status ON bets (status);
CREATE INDEX idx_settlements_market ON settlements (market_id);
CREATE INDEX idx_settlements_user ON settlements (user_id);
CREATE INDEX idx_payment_transactions_user ON payment_transactions (user_id);
CREATE INDEX idx_payment_transactions_provider_ref ON payment_transactions (provider, provider_reference);
CREATE INDEX idx_audit_logs_user ON audit_logs (user_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);
CREATE INDEX idx_token_blacklist_jti ON token_blacklist (token_jti);
CREATE INDEX idx_token_blacklist_expires_at ON token_blacklist (expires_at);

-- Insert default Nigeria country
INSERT INTO countries (name, code, currency_code, currency_symbol, config)
VALUES ('Nigeria', 'NGA', 'NGN', '₦', '{
  "contract_unit": 100,
  "min_bet": 100,
  "max_bet": 50000,
  "kyc_required": true
}');

-- Insert default categories for Nigeria
INSERT INTO categories (country_id, name, slug, description, sort_order)
SELECT c.id,
       category_data.name,
       category_data.slug,
       category_data.description,
       category_data.sort_order
FROM countries c,
     (VALUES ('Artificial Intelligence', 'ai', 'AI model releases, capability benchmarks, company announcements', 1),
             ('Tech Events', 'tech-events', 'WWDC, Google I/O, product launches, conference announcements', 2),
             ('Cryptocurrency', 'crypto', 'Price predictions, regulatory decisions, platform updates', 3),
             ('Nigerian Tech', 'nigerian-tech', 'Local startup funding, policy changes, market developments',
              4)) AS category_data(name, slug, description, sort_order)
WHERE c.code = 'NGA';
