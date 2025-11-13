
-- name: AssignAmountToCategory :one
INSERT INTO assignments (month, category_id, assigned)
VALUES (
  DATE_TRUNC('month', sqlc.arg('month_id')::timestamp),
  sqlc.arg('category_id'),
  sqlc.arg('amount')
)
ON CONFLICT (month, category_id)
DO UPDATE
SET assigned = assignments.assigned + EXCLUDED.assigned
RETURNING *;

-- name: GetMonthReport :one
SELECT SUM(assigned) AS assigned, SUM(activity) AS activity, SUM(balance) AS balance
FROM category_reports mr
WHERE mr.month = $1;

-- name: GetMonthCategoryReports :many
SELECT * FROM category_reports cr
WHERE cr.month = $1;

-- name: GetMonthCategoryReport :one
SELECT * FROM category_reports cr
WHERE cr.month = $1 AND cr.category_id = $2;

-- name: DeleteMonthAssignmentForCat :exec
DELETE FROM assignments
WHERE $1 = month AND $2 = category_id;

