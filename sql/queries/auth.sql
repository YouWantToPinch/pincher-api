/* TOKEN CRUD */

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES (
    sqlc.arg('token'),
    NOW(),
    NOW(),
    sqlc.arg('user_id'),
    sqlc.arg('expires_at'),
    NULL
)
RETURNING *;

-- name: GetUserByRefreshToken :one
SELECT users.*
FROM users
JOIN refresh_tokens ON users.id = refresh_tokens.user_id
WHERE refresh_tokens.token = $1
AND revoked_at IS NULL
AND expires_at > NOW();

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = NOW(), updated_at = NOW()
WHERE token = $1
AND revoked_at IS NULL;
