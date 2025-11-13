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

