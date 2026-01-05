/* USER CRUD */

-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, username, hashed_password)
VALUES (
    gen_random_uuid(),
    DEFAULT,
    DEFAULT,
    $1,
    $2
)
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1;

-- name: UpdateUserCredentials :one
UPDATE users
SET updated_at = NOW(), username = $2, hashed_password = $3
WHERE id = $1
RETURNING *;

-- name: DeleteUserByID :exec
DELETE
FROM users
WHERE id = $1;
