-- USER ACTIONS

-- name: GetUserCount :one
SELECT COUNT(*) from users;

-- name: GetAllUsers :many
SELECT * FROM users;

-- name: DeleteUsers :exec
DELETE FROM users;