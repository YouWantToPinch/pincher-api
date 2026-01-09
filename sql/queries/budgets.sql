-- name: CreateBudget :one
INSERT INTO budgets (id, created_at, updated_at, admin_id, name, notes)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2,
    $3
)
RETURNING *;

-- name: GetUserBudgets :many
SELECT budgets.*
FROM budgets
JOIN budgets_users
  ON budgets.id = budgets_users.budget_id
WHERE budgets_users.user_id = $1
  AND (
    sqlc.arg('roles')::text[] IS NULL
    OR cardinality(sqlc.arg('roles')::text[]) = 0
    OR budgets_users.member_role = ANY(sqlc.arg('roles')::text[])
  );

-- name: GetBudgetByID :one
SELECT *
FROM budgets
WHERE id = $1;

-- name: GetBudgetMemberRole :one
SELECT member_role
FROM budgets_users
WHERE budget_id = $1 AND user_id = $2;

-- name: AssignBudgetMemberWithRole :one
INSERT INTO budgets_users (created_at, updated_at, budget_id, user_id, member_role)
VALUES (
    NOW(),
    NOW(),
    $1,
    $2,
    $3
)
ON CONFLICT (budget_id, user_id) DO UPDATE
SET updated_at = EXCLUDED.updated_at, member_role = EXCLUDED.member_role
RETURNING *;

-- name: GetBudgetCapital :one
SELECT CAST(COALESCE(SUM(td.total_amount), 0) AS BIGINT) AS total
FROM transaction_details td
JOIN transactions t ON td.id = t.id
WHERE t.budget_id = $1;

-- name: UpdateBudget :one
UPDATE budgets
SET updated_at = NOW(), name = $2, notes = $3
WHERE id = $1
RETURNING *;

-- name: RevokeBudgetMembership :exec
DELETE
FROM budgets_users
WHERE budget_id = $1
    AND user_id = $2;

-- name: DeleteBudget :exec
DELETE
FROM budgets
WHERE id = $1;


-- RESOURCE ID RETRIEVAL

-- name: GetBudgetAccountIDByName :one
SELECT id
FROM accounts
WHERE name = sqlc.arg('account_name')
AND budget_id = sqlc.arg('budget_id');

-- name: GetBudgetCategoryIDByName :one
SELECT id
FROM categories
WHERE name = sqlc.arg('category_name')
AND budget_id = sqlc.arg('budget_id');

-- name: GetBudgetPayeeIDByName :one
SELECT id
FROM payees
WHERE name = sqlc.arg('payee_name')
AND budget_id = sqlc.arg('budget_id');

-- name: GetBudgetGroupIDByName :one
SELECT id
FROM groups
WHERE name = sqlc.arg('group_name')
AND budget_id = sqlc.arg('budget_id');

