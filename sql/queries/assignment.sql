
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
SELECT
    COALESCE(SUM(assigned), 0)::bigint AS assigned,
    COALESCE(SUM(activity), 0)::bigint AS activity,
    COALESCE(SUM(balance), 0)::bigint AS balance
FROM category_reports mr
WHERE mr.month = date_trunc('month', sqlc.arg('month_id')::date);

-- name: GetMonthCategoryReports :many
SELECT * FROM category_reports cr
JOIN categories ON cr.category_id = categories.id
WHERE cr.month = date_trunc('month', sqlc.arg('month_id')::date) 
  AND categories.budget_id = sqlc.arg('budget_id')::uuid;

-- name: GetMonthCategoryReport :one
SELECT * FROM category_reports cr
JOIN categories ON cr.category_id = categories.id
WHERE cr.month = date_trunc('month', sqlc.arg('month_id')::date) 
  AND cr.category_id = sqlc.arg('category_id')::uuid
  AND categories.budget_id = sqlc.arg('budget_id')::uuid;

-- name: DeleteMonthAssignmentForCat :exec
DELETE FROM assignments
WHERE $1 = month AND $2 = category_id;

