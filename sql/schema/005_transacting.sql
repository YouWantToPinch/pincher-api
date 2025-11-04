-- +goose Up
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    budget_id UUID NOT NULL,
    account_type TEXT NOT NULL,
    name TEXT NOT NULL,
    notes TEXT,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
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
    notes TEXT NOT NULL,
    cleared BOOLEAN NOT NULL DEFAULT FALSE,
    FOREIGN KEY (budget_id) REFERENCES budgets(id)
      ON DELETE CASCADE,
    FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE TABLE account_transfers (
  id UUID PRIMARY KEY,
  from_transaction_id UUID NOT NULL,
  to_transaction_id UUID NOT NULL,
  UNIQUE (from_transaction_id, to_transaction_id),
  FOREIGN KEY (from_transaction_id) REFERENCES transactions(id)
    ON DELETE CASCADE,
  FOREIGN KEY (to_transaction_id) REFERENCES transactions(id)
    ON DELETE CASCADE
);

CREATE TABLE payees (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    budget_id UUID NOT NULL,
    name VARCHAR(32) NOT NULL,
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
  -- FOREIGN KEY (category_id) REFERENCES categories(id)
);

CREATE VIEW transactions_view AS
SELECT 
  t.id,
  t.transaction_type,
  t.transaction_date,
  COALESCE(
    p.name,
    (
      SELECT a.name
      FROM account_transfers at
      JOIN transactions t2 ON (
        (at.from_transaction_id = t.id AND t2.id = at.to_transaction_id)
        OR
        (at.to_transaction_id = t.id AND t2.id = at.from_transaction_id)
      )
      JOIN accounts a ON t2.account_id = a.id
      LIMIT 1
    ),
    'Transfer'
  ) AS payee,
  t.payee_id,
  t.notes,
  t.budget_id,
  t.account_id,
  t.logger_id,
  SUM(ts.amount)::bigint AS total_amount,
  jsonb_object_agg(COALESCE(c.name, 'Uncategorized'), ts.amount) AS splits,
  t.cleared
FROM transactions t
JOIN transaction_splits ts ON t.id = ts.transaction_id
LEFT JOIN categories c ON ts.category_id = c.id
LEFT JOIN payees p ON t.payee_id = p.id
GROUP BY
    t.id,
    t.transaction_type,
    t.transaction_date,
    p.name,
    t.payee_id,
    t.notes,
    t.account_id,
    t.logger_id,
    t.cleared;

-- +goose Down
DROP VIEW transactions_view;
DROP TABLE transaction_splits;
DROP TABLE payees;
DROP TABLE account_transfers;
DROP TABLE transactions;
DROP TABLE accounts;
