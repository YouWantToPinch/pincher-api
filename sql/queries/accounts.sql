-- name: AddAccount :one
INSERT INTO accounts (id, created_at, updated_at, budget_id, account_type, name, notes, is_deleted)
VALUES (
    gen_random_uuid(),
    DEFAULT,
    DEFAULT,
    $1,
    $2,
    $3,
    $4,
    DEFAULT
)
RETURNING *;

-- name: GetAccountsFromBudget :many
SELECT *
FROM accounts
WHERE accounts.budget_id = $1;

-- name: GetAccountByID :one
SELECT *
FROM accounts
WHERE accounts.id = $1;

-- name: GetBudgetAccountCapital :one
SELECT CAST(COALESCE(SUM(transactions_view.total_amount), 0) AS BIGINT) AS total
FROM transactions_view
WHERE transactions_view.account_id = $1;

-- name: RestoreAccount :exec
UPDATE accounts
SET accounts.is_deleted = FALSE
WHERE accounts.id = $1;

-- name: UpdateAccount :one
UPDATE accounts
SET updated_at = NOW(), account_type = $2, name = $3, notes = $4
WHERE id = $1
RETURNING *;

-- name: DeleteAccountSoft :exec
UPDATE accounts
SET accounts.is_deleted = TRUE
WHERE accounts.id = $1;

-- name: DeleteAccountHard :exec
DELETE
FROM accounts
WHERE accounts.id = $1
    AND accounts.is_deleted = TRUE;
