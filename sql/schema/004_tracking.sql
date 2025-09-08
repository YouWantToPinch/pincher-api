-- +goose Up
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    user_id UUID NOT NULL,
    account_type TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE

);

CREATE TABLE transactions (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    user_id UUID NOT NULL,
    account UUID NOT NULL,
    datetime TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    payees TEXT NOT NULL,
    notes TEXT NOT NULL,
    amount INTEGER,
    split_amount TEXT NOT NULL,
    cleared BOOLEAN NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- +goose Down
DROP TABLE transactions;
DROP TABLE accounts;