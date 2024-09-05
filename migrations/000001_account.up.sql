-- Account table to store the single user's balance
CREATE TABLE account (
                         id INTEGER PRIMARY KEY DEFAULT 1,
                         balance DECIMAL(15, 5) NOT NULL DEFAULT 0.00 CHECK (balance >= 0),
                         updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                         version INTEGER NOT NULL DEFAULT 0
);


-- Insert the single user
INSERT INTO account (balance, version) VALUES (0.00, 0);
