-- +goose Up
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    budget_id UUID NOT NULL,
    account_type TEXT NOT NULL,
    name VARCHAR(50) NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE(budget_id, name),
    FOREIGN KEY (budget_id) REFERENCES budgets(id)
        ON DELETE CASCADE
);

CREATE TABLE transactions (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    budget_id UUID NOT NULL,
    logger_id UUID NOT NULL,
    account_id UUID NOT NULL,
    transaction_type VARCHAR(15) NOT NULL,
    transaction_date TIMESTAMP NOT NULL DEFAULT (DATE_TRUNC('day', NOW() AT TIME ZONE 'utc')),
    payee_id UUID NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    cleared BOOLEAN NOT NULL DEFAULT FALSE,
    FOREIGN KEY (budget_id) REFERENCES budgets(id)
      ON DELETE CASCADE,
    FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE TABLE account_transfers (
  from_transaction_id UUID NOT NULL,
  to_transaction_id UUID NOT NULL,
  FOREIGN KEY (from_transaction_id) REFERENCES transactions(id)
    ON DELETE CASCADE,
  FOREIGN KEY (to_transaction_id) REFERENCES transactions(id)
    ON DELETE CASCADE,
  PRIMARY KEY (from_transaction_id, to_transaction_id)
);

CREATE TABLE payees (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    budget_id UUID NOT NULL,
    name VARCHAR(50) NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    UNIQUE(budget_id, name),
    FOREIGN KEY (budget_id) REFERENCES budgets(id)
      ON DELETE CASCADE
);

CREATE TABLE transaction_splits (
  id UUID PRIMARY KEY,
  transaction_id UUID NOT NULL,
  category_id UUID,
  amount BIGINT NOT NULL,
  UNIQUE (transaction_id, category_id),
  FOREIGN KEY (transaction_id) REFERENCES transactions(id)
    ON DELETE CASCADE
);

CREATE VIEW transaction_details AS
SELECT 
  t.id,
  t.transaction_date,
  t.transaction_type,
  t.notes,
  COALESCE(
    p.name,
    'Transfer'
  ) AS payee_name,
  b.name AS budget_name,
  a.name AS account_name,
  u.username AS logger_name,
  SUM(ts.amount)::bigint AS total_amount,
  jsonb_object_agg(COALESCE(c.name, 'Uncategorized'), ts.amount) AS splits,
  t.cleared
FROM transactions t
JOIN transaction_splits ts ON t.id = ts.transaction_id
LEFT JOIN categories c ON ts.category_id = c.id
LEFT JOIN payees p ON t.payee_id = p.id
LEFT JOIN accounts a ON t.account_id = a.id
LEFT JOIN users u ON t.logger_id = u.id
LEFT JOIN budgets b ON t.budget_id = b.id
GROUP BY
    t.id,
    t.transaction_date,
    t.transaction_type,
    t.notes,
    p.name,
    a.name,
    u.username,
    b.name
ORDER BY
    t.transaction_date DESC;

-- +goose Down
DROP VIEW transaction_details;
DROP TABLE transaction_splits;
DROP TABLE payees;
DROP TABLE account_transfers;
DROP TABLE transactions;
DROP TABLE accounts;
