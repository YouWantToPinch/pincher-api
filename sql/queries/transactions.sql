-- name: LogTransaction :one
INSERT INTO transactions (id, created_at, updated_at, budget_id, logger_id, account_id, transaction_date, payee_id, notes, cleared)
VALUES (
    gen_random_uuid(),
    DEFAULT,
    DEFAULT,
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
)
RETURNING *;

-- name: LogTransactionSplit :one
INSERT INTO transaction_splits (id, transaction_id, category_id, amount)
VALUES (
    gen_random_uuid(),
    $1,
    $2,
    $3
)
RETURNING *;

-- name: GetTransactionsFromView :many
SELECT *
FROM transactions_view t
WHERE
    t.budget_id = sqlc.arg('budget_id')
    AND (sqlc.arg('account_id')::uuid = '00000000-0000-0000-0000-000000000000' OR t.account_id = sqlc.arg('account_id')::uuid)
    AND (
        sqlc.arg('category_id')::uuid = '00000000-0000-0000-0000-000000000000'
        OR EXISTS (
            SELECT 1
            FROM transaction_splits ts
            WHERE ts.transaction_id = t.id AND ts.category_id = sqlc.arg('category_id')::uuid
        )
    )
    AND (sqlc.arg('payee_id')::uuid = '00000000-0000-0000-0000-000000000000' OR t.payee_id = sqlc.arg('payee_id')::uuid)
    AND (
        (sqlc.arg('start_date')::date = '0001-01-01' AND sqlc.arg('end_date')::date = '0001-01-01')
        OR
        (t.transaction_date >= sqlc.arg('start_date')::date AND t.transaction_date <= sqlc.arg('end_date')::date)
    )
ORDER BY t.transaction_date DESC;

-- name: GetTransactions :many
SELECT *
FROM transactions t
WHERE
    t.budget_id = sqlc.arg('budget_id')
    AND (sqlc.arg('account_id')::uuid = '00000000-0000-0000-0000-000000000000' OR t.account_id = sqlc.arg('account_id')::uuid)
    AND (
        sqlc.arg('category_id')::uuid = '00000000-0000-0000-0000-000000000000'
        OR EXISTS (
            SELECT 1
            FROM transaction_splits ts
            WHERE ts.transaction_id = t.id AND ts.category_id = sqlc.arg('category_id')::uuid
        )
    )
    AND (sqlc.arg('payee_id')::uuid = '00000000-0000-0000-0000-000000000000' OR t.payee_id = sqlc.arg('payee_id')::uuid)
    AND (
        (sqlc.arg('start_date')::date = '0001-01-01' AND sqlc.arg('end_date')::date = '0001-01-01')
        OR
        (t.transaction_date >= sqlc.arg('start_date')::date AND t.transaction_date <= sqlc.arg('end_date')::date)
    )
ORDER BY t.transaction_date DESC;

-- name: GetSplitsByTransactionID :many
SELECT *
FROM transaction_splits
WHERE transaction_splits.id = $1;

-- name: GetTransactionByID :one
SELECT *
FROM transactions
WHERE id = $1;

-- name: GetTransactionFromViewByID :one
SELECT *
FROM transactions_view
WHERE id = $1;

-- name: DeleteTransaction :exec
DELETE
FROM transactions
WHERE transactions.id = $1;
