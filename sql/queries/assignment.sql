
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

-- name: ReassignAmountToCategory :one
INSERT INTO assignments (month, category_id, assigned)
VALUES (
  DATE_TRUNC('month', @month_id::timestamp),
  @category_id,
  @amount
)
ON CONFLICT (month, category_id)
DO UPDATE
SET assigned = EXCLUDED.assigned
RETURNING *,
  (assigned - (SELECT assigned 
  FROM assignments 
  WHERE month = DATE_TRUNC('month', @month_id::timestamp) 
    AND category_id = @category_id)) 
  AS change_in_amount;

-- name: DeleteMonthAssignmentForCat :exec
DELETE FROM assignments
WHERE $1 = month AND $2 = category_id;

