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
            WHEN tr1.transaction_type ILIKE '%DEPOSIT%' AND key ILIKE '%UNCATEGORIZED%' THEN NULL
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
    ) AS splits
FROM tr1;

-- name: LogAccountTransfer :one
INSERT INTO account_transfers (from_transaction_id, to_transaction_id)
VALUES (
    sqlc.arg('from_transaction_id'),
    sqlc.arg('to_transaction_id')
)
RETURNING *;

-- name: GetTransactionDetails :many
SELECT *
FROM transaction_details td
JOIN transactions t ON td.id = t.id
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
WHERE transaction_id = $1;

-- name: GetTransactionByID :one
SELECT *
FROM transactions
WHERE id = $1;

-- name: GetTransactionDetailsByID :one
SELECT *
FROM transaction_details
WHERE id = $1;

-- name: UpdateTransaction :one
WITH
updated_txn AS (
    UPDATE transactions t
    SET
        updated_at = NOW(),
        account_id = sqlc.arg('account_id'),
        transaction_type = sqlc.arg('transaction_type'),
        transaction_date = sqlc.arg('transaction_date'),
        payee_id = sqlc.arg('payee_id'),
        notes = sqlc.arg('notes'),
        cleared = sqlc.arg('cleared')
    WHERE t.id = sqlc.arg('transaction_id')
    RETURNING *
),

-- Clear out existing splits for this transaction
deleted_splits AS (
    DELETE FROM transaction_splits
    WHERE transaction_id = sqlc.arg('transaction_id')
),

-- Insert fresh splits from the provided JSON
inserted_splits AS (
    INSERT INTO transaction_splits (id, transaction_id, category_id, amount)
    SELECT
        gen_random_uuid(),
        updated_txn.id,
        CASE
            WHEN updated_txn.transaction_type ILIKE '%TRANSFER%' THEN NULL
            WHEN updated_txn.transaction_type ILIKE '%DEPOSIT%' AND key ILIKE '%UNCATEGORIZED%' THEN NULL
            ELSE key::uuid
        END,
        value::integer
    FROM updated_txn, json_each_text(sqlc.arg('amounts')::json)
    RETURNING *
)
SELECT
    updated_txn.*,
    (
        SELECT json_agg(inserted_splits.*)
        FROM inserted_splits
    ) AS splits
FROM updated_txn;

-- name: DeleteTransaction :exec
DELETE
FROM transactions
WHERE id = $1;
