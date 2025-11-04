
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
SELECT SUM(assigned) AS assinged, SUM(activity) AS activity, SUM(balance) AS balance
FROM month_report mr
WHERE mr.month = $1;

-- name: GetMonthCategoryReports :many
SELECT * FROM month_report mr
WHERE mr.month = $1;

-- name: GetMonthCategoryReport :one
SELECT * FROM month_report mr
WHERE mr.month = $1 AND mr.category_id = $2;

-- name: DeleteMonthAssignmentForCat :exec
DELETE FROM assignments
WHERE $1 = assignments.month AND $2 = assignments.category_id;
