
-- name: AssignAmountToCategory :one
INSERT INTO assignments (month, category_id, assigned)
VALUES (
  DATE_TRUNC('month', @month_id::timestamp),
  @category_id,
  @amount
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
WHERE mr.month = date_trunc('month', @month_id::date)
  AND mr.budget_id = @budget_id::uuid;

-- name: GetMonthCategoryReports :many
SELECT * FROM category_reports cr
JOIN categories ON cr.category_id = categories.id
WHERE cr.month = date_trunc('month', @month_id::date)
  AND categories.budget_id = @budget_id::uuid;

-- name: GetMonthCategoryReport :one
SELECT * FROM category_reports cr
JOIN categories ON cr.category_id = categories.id
WHERE cr.month = date_trunc('month', @month_id::date)
  AND cr.category_id = @category_id::uuid
  AND categories.budget_id = @budget_id::uuid;

-- name: DeleteMonthAssignmentForCat :exec
DELETE FROM assignments
WHERE $1 = month AND $2 = category_id;

