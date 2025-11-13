-- name: CreatePayee :one
INSERT INTO payees (id, created_at, updated_at, budget_id, name, notes)
VALUES (
    gen_random_uuid(),
    DEFAULT,
    DEFAULT,
    $1,
    $2,
    $3
)
RETURNING *;

-- name: GetBudgetPayees :many
SELECT *
FROM payees
WHERE payees.budget_id = $1;

-- name: GetPayeeByID :one
SELECT *
FROM payees
WHERE payees.id = $1;

-- name: UpdatePayee :one
UPDATE payees
SET updated_at = NOW(), name = $2, notes = $3
WHERE id = $1
RETURNING *;

-- name: DeletePayee :exec
DELETE
FROM payees
WHERE payees.id = $1;
