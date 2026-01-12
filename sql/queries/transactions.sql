-- name: LogTransaction :one
INSERT INTO transactions (
    id, created_at, updated_at, budget_id, logger_id,
    account_id, transaction_type, transaction_date,
    payee_id, notes, cleared)
VALUES (
    gen_random_uuid(),
    DEFAULT,
    DEFAULT,
    sqlc.arg('budget_id'),
    sqlc.arg('logger_id'),
    sqlc.arg('account_id'),
    sqlc.arg('transaction_type'),
    sqlc.arg('transaction_date'),
    sqlc.arg('payee_id'),
    sqlc.arg('notes'),
    sqlc.arg('cleared')
)
RETURNING *;

-- name: LogTransactionSplits :many
INSERT INTO transaction_splits (id, transaction_id, category_id, amount)
SELECT
  gen_random_uuid(),
  sqlc.arg('transaction_id')::uuid,
  CASE
    WHEN t.transaction_type ILIKE '%TRANSFER%' THEN NULL
    WHEN t.transaction_type ILIKE '%DEPOSIT%' AND key ILIKE '%UNCATEGORIZED%' THEN NULL
    ELSE key::uuid
  END,
  value::integer
FROM json_each_text(sqlc.arg('amounts')::json)
JOIN transactions t ON t.id = sqlc.arg('transaction_id')::uuid
RETURNING *;

-- name: LogAccountTransfer :one
INSERT INTO account_transfers (from_transaction_id, to_transaction_id)
VALUES (
    sqlc.arg('from_transaction_id')::uuid,
    sqlc.arg('to_transaction_id')::uuid
)
RETURNING *;

-- HACK:
-- When it comes to nullable values, sqlc seems to have
-- a difficult time inferring any sort of nullability
-- on query parameters. This zero-value approach
-- ensures that the zero-value UUIDs and timestamps
-- passed to the query are properly compared.

-- name: GetTransactionDetails :many
SELECT td.*
FROM transaction_details td
JOIN transactions t ON td.id = t.id
WHERE
  budget_id = sqlc.arg('budget_id')::uuid
  AND (
      sqlc.arg('account_id')::uuid = '00000000-0000-0000-0000-000000000000'
      OR t.account_id = sqlc.arg('account_id')::uuid
    )
  AND (
    sqlc.arg('payee_id')::uuid = '00000000-0000-0000-0000-000000000000'
    OR t.payee_id = sqlc.arg('payee_id')::uuid
  )
  AND (
    sqlc.arg('category_id')::uuid = '00000000-0000-0000-0000-000000000000'
    OR EXISTS (
      SELECT 1
      FROM transaction_splits ts
      WHERE ts.transaction_id = t.id AND ts.category_id = sqlc.arg('category_id')::uuid
    )
  )
  AND (
    (sqlc.arg('start_date')::date = '0001-01-01' AND sqlc.arg('end_date')::date = '0001-01-01')
    OR (t.transaction_date BETWEEN sqlc.arg('start_date')::date AND sqlc.arg('end_date')::date)
  )
ORDER BY t.transaction_date DESC;

-- name: GetTransactions :many
SELECT t.*
FROM transactions t
WHERE
  budget_id = sqlc.arg('budget_id')::uuid
  AND (
      sqlc.arg('account_id')::uuid = '00000000-0000-0000-0000-000000000000'
      OR t.account_id = sqlc.arg('account_id')::uuid
    )
  AND (
    sqlc.arg('payee_id')::uuid = '00000000-0000-0000-0000-000000000000'
    OR t.payee_id = sqlc.arg('payee_id')::uuid
  )
  AND (
    sqlc.arg('category_id')::uuid = '00000000-0000-0000-0000-000000000000'
    OR EXISTS (
      SELECT 1
      FROM transaction_splits ts
      WHERE ts.transaction_id = t.id AND ts.category_id = sqlc.arg('category_id')::uuid
    )
  )
  AND (
    (sqlc.arg('start_date')::date = '0001-01-01' AND sqlc.arg('end_date')::date = '0001-01-01')
    OR (t.transaction_date BETWEEN sqlc.arg('start_date')::date AND sqlc.arg('end_date')::date)
  )
ORDER BY t.transaction_date DESC;

-- name: GetSplitsByTransactionID :many
SELECT *
FROM transaction_splits
WHERE transaction_id = $1;

-- name: GetTransactionByID :one
SELECT *
FROM transactions
WHERE id = $1;

-- name: GetTransactionDetailsByID :one
SELECT *
FROM transaction_details
WHERE id = $1;

-- name: UpdateTransaction :exec
UPDATE transactions t
SET
  updated_at = NOW(),
  account_id = sqlc.arg('account_id'),
  transaction_type = sqlc.arg('transaction_type'),
  transaction_date = sqlc.arg('transaction_date'),
  payee_id = sqlc.arg('payee_id'),
  notes = sqlc.arg('notes'),
  cleared = sqlc.arg('cleared')
WHERE t.id = sqlc.arg('transaction_id');

-- name: DeleteTransactionSplits :exec
  DELETE FROM transaction_splits
  WHERE transaction_id = sqlc.arg('transaction_id');

-- name: GetLinkedTransaction :one
SELECT t.*
FROM transactions t
JOIN account_transfers at
  ON (t.id = at.to_transaction_id AND at.from_transaction_id = $1)
     OR (t.id = at.from_transaction_id AND at.to_transaction_id = $1);

-- name: DeleteTransaction :exec
DELETE
FROM transactions
WHERE id = $1;
