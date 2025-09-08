/* USER CRUD */

-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, username, hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE users.username = $1;

-- name: UpdateUserCredentials :one
UPDATE users
SET updated_at = NOW(), username = $2, hashed_password = $3
WHERE id = $1
RETURNING *;

-- name: DeleteUsers :exec
DELETE FROM users;

-- name: DeleteUserByID :exec
DELETE
FROM users
WHERE id = $1;
