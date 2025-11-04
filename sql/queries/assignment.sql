
-- name: AssignAmountToCategory :one
INSERT INTO assignments (month, category_id, assigned)
VALUES (
  sqlc.arg('month_id'),
  sqlc.arg('category_id'),
  sqlc.arg('assigned')
)
ON CONFLICT (month, category_id)
DO UPDATE
SET assigned = assignments.assigned + EXCLUDED.assigned
RETURNING *;

-- name: GetMonthReport :many
SELECT * FROM month_report ar
WHERE ar.month = $1;

-- name: GetMonthCategory :one
SELECT * FROM month_report ar
WHERE ar.month = $1 AND ar.category_id = $2;

-- name: DeleteMonthAssignmentForCat :exec
DELETE FROM assignments
WHERE $1 = assignments.month AND $2 = assignments.category_id;
