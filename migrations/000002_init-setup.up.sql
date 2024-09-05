CREATE TABLE IF NOT EXISTS transactions
(
    id             SERIAL PRIMARY KEY,
    transaction_id VARCHAR(255) UNIQUE NOT NULL,
    account_id     INTEGER REFERENCES account (id),
    amount         DECIMAL(15, 5)      NOT NULL,
    state          VARCHAR(10)         NOT NULL CHECK (state IN ('win', 'lost')),
    source_type    VARCHAR(20)         NOT NULL,
    processed_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_canceled    BOOLEAN                  DEFAULT FALSE
);