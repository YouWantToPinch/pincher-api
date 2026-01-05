-- name: CreateCategory :one
INSERT INTO categories (id, created_at, updated_at, budget_id, group_id, name, notes)
VALUES (
    gen_random_uuid(),
    DEFAULT,
    DEFAULT,
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetCategoryByID :one
SELECT *
FROM categories
WHERE id = $1;

-- name: GetCategories :many
SELECT *
FROM categories c
WHERE c.budget_id = sqlc.arg('budget_id')
  AND ( -- TODO: Lesson learned. Separate stuff like this into separate queries! This is just messy.
    sqlc.arg('group_id')::uuid = '00000000-0000-0000-0000-000000000000'
    OR c.group_id = sqlc.arg('group_id')::uuid
  );

-- name: UpdateCategory :one
UPDATE categories
SET updated_at = NOW(), group_id = $2, name = $3, notes = $4
WHERE id = $1
RETURNING *;

-- name: DeleteCategoryByID :exec
DELETE
FROM categories
WHERE id = $1;
