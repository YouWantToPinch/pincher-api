-- name: LogTransaction :one
WITH
tr1 AS (
    INSERT INTO transactions (
        id, created_at, updated_at, budget_id, logger_id,
        account_id, transaction_type, transaction_date,
        payee_id, notes, cleared
    )
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
    RETURNING *
),
insert_splits AS (
    INSERT INTO transaction_splits (id, transaction_id, category_id, amount)
    SELECT
        gen_random_uuid(),
        tr1.id,
        CASE
            WHEN tr1.transaction_type ILIKE '%TRANSFER%' THEN NULL
            ELSE key::uuid
        END,
        value::integer
    FROM tr1, json_each_text(sqlc.arg('amounts')::json)
    RETURNING *
)
SELECT 
    tr1.*,
    (
        SELECT json_agg(insert_splits.*)
        FROM insert_splits
        WHERE insert_splits.transaction_id = tr1.id
    ) AS splits
FROM tr1;

-- name: LogAccountTransfer :one
INSERT INTO account_transfers (id, from_transaction_id, to_transaction_id)
VALUES (
    gen_random_uuid(),
    sqlc.arg('from_transaction_id'),
    sqlc.arg('to_transaction_id')
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
