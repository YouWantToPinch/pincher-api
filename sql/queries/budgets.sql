/* BUDGET CRUD */

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
(
  SELECT budgets.*
  FROM budgets
  JOIN budgets_users
    ON budgets.id = budgets_users.budget_id
  WHERE budgets_users.user_id = $1
    AND (sqlc.arg('roles')::text[] IS NULL OR budgets_users.member_role = ANY(sqlc.arg('roles')::text[]))
)
UNION
(
  SELECT budgets.*
  FROM budgets
  WHERE budgets.admin_id = $1
);

-- name: GetBudgetByID :one
SELECT *
FROM budgets
WHERE budgets.id = $1;

-- name: GetBudgetMemberRole :one
SELECT member_role
FROM budgets_users
WHERE budgets_users.budget_id = $1 AND budgets_users.user_id = $2;

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
SELECT CAST(COALESCE(SUM(transactions_view.total_amount), 0) AS BIGINT) AS total
FROM transactions_view
WHERE transactions_view.budget_id = $1;

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
WHERE budgets.id = $1;

/* GROUP CRUD */

-- name: CreateGroup :one
INSERT INTO groups (id, created_at, updated_at, budget_id, name, notes)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2,
    $3
)
RETURNING *;

-- name: GetGroupByID :one
SELECT *
FROM groups
WHERE groups.budget_id = $1
    AND groups.id = $2;

-- name: GetGroupsByBudgetID :many
SELECT *
FROM groups
WHERE groups.budget_id = $1;

-- name: GetGroupByBudgetIDAndName :one
SELECT *
FROM groups
WHERE groups.name = $1 AND groups.budget_id = $2;

-- name: UpdateGroup :one
UPDATE groups
SET updated_at = NOW(), name = $2, notes = $3
WHERE id = $1
RETURNING *;

-- name: DeleteGroupByID :exec
DELETE
FROM groups
WHERE groups.id = $1;

/* CATEGORY CRUD */

-- name: CreateCategory :one
INSERT INTO categories (id, created_at, updated_at, budget_id, group_id, name, notes)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetCategoryByID :one
SELECT *
FROM categories
WHERE categories.id = $1;

-- name: GetCategories :many
SELECT *
FROM categories c
WHERE c.budget_id = sqlc.arg('budget_id')
  AND (
    sqlc.arg('group_id')::uuid IS NULL
    OR c.group_id = sqlc.arg('group_id')
  );

-- name: UpdateCategory :one
UPDATE categories
SET updated_at = NOW(), group_id = $2, name = $3, notes = $4
WHERE id = $1
RETURNING *;

-- name: DeleteCategoryByID :exec
DELETE
FROM categories
WHERE categories.id = $1;
