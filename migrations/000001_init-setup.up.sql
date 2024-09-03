CREATE TABLE IF NOT EXISTS transactions
(
    id             SERIAL PRIMARY KEY,
    transaction_id VARCHAR(100) UNIQUE NOT NULL,
    source_type    VARCHAR(20)         NOT NULL CHECK (source_type IN ('client', 'game', 'server', 'payment')),
    state          VARCHAR(10)         NOT NULL CHECK (state IN ('win', 'lost')),
    amount         DECIMAL(15, 5)      NOT NULL,
    is_processed   BOOLEAN             NOT NULL DEFAULT false,
    is_canceled    BOOLEAN             NOT NULL DEFAULT false,
    created_at     TIMESTAMP WITH TIME ZONE     DEFAULT CURRENT_TIMESTAMP,
    processed_at   TIMESTAMP WITH TIME ZONE,
    canceled_at    TIMESTAMP WITH TIME ZONE
);
