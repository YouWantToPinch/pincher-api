/* GROUP CRUD */

-- name: CreateGroup :one
INSERT INTO groups (id, created_at, updated_at, user_id, name, notes)
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
WHERE groups.id = $1;

-- name: GetGroupsByUserID :many
SELECT *
FROM groups
WHERE groups.user_id = $1;

-- name: GetGroupByUserIDAndName :one
SELECT *
FROM groups
WHERE groups.name = $1 AND groups.user_id = $2;

-- name: DeleteGroupByID :exec
DELETE
FROM groups
WHERE groups.id = $1;

/* CATEGORY CRUD */

-- name: CreateCategory :one
INSERT INTO categories (id, created_at, updated_at, user_id, group_id, name, notes)
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

-- name: GetCategoriesByUserID :many
SELECT *
FROM categories
WHERE categories.user_id = $1;

-- name: GetCategoriesByGroup :many
SELECT *
FROM categories
WHERE categories.group_id = $1;

-- name: AssignCategoryToGroup :one
UPDATE categories
SET updated_at = NOW(), group_id = $2
WHERE id = $1
RETURNING *;

-- name: DeleteCategoryByID :exec
DELETE
FROM categories
WHERE categories.id = $1;