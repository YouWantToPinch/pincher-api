-- name: GetMonthReport :one
SELECT 
  @month_id::date AS month,
  COALESCE(SUM(assigned), 0)::bigint AS assigned,
  COALESCE(SUM(activity), 0)::bigint AS activity,
  COALESCE(SUM(balance), 0)::bigint AS balance
FROM rep.get_category_reports_gate(
  @budget_id::uuid, 
  @month_id::date
)
GROUP BY month;

-- name: GetMonthCategoryReports :many
SELECT 
  @month_id::date AS month,
  category_id::uuid AS category_id,
  COALESCE(g.id, '00000000-0000-0000-0000-000000000000')::uuid AS group_id,
  category_name::text AS category_name,
  COALESCE(assigned, 0)::bigint AS assigned,
  COALESCE(activity, 0)::bigint AS activity,
  COALESCE(balance, 0)::bigint AS balance
FROM rep.get_category_reports_gate(
  @budget_id::uuid, 
  @month_id::date
) AS r
LEFT JOIN categories c ON r.category_id = c.id
LEFT JOIN groups g ON c.group_id = g.id
ORDER BY category_name;

-- name: GetMonthCategoryReport :one
SELECT 
  @month_id::date AS month,
  category_id::uuid AS category_id,
  COALESCE(g.id, '00000000-0000-0000-0000-000000000000')::uuid AS group_id,
  category_name::text AS category_name,
  COALESCE(assigned, 0)::bigint AS assigned,
  COALESCE(activity, 0)::bigint AS activity,
  COALESCE(balance, 0)::bigint AS balance
FROM rep.get_category_reports_gate(
  @budget_id::uuid, 
  @month_id::date
) AS r
LEFT JOIN categories c ON r.category_id = c.id
LEFT JOIN groups g ON c.group_id = g.id
WHERE r.category_id = @category_id::uuid
LIMIT 1;

-- name: GetMonthGroupReports :many
SELECT 
  @month_id::date AS month,
  COALESCE(g.name, 'Ungrouped')::text AS group_name,
  COALESCE(g.id, '00000000-0000-0000-0000-000000000000')::uuid AS group_id,
  COALESCE(SUM(r.assigned), 0)::bigint AS assigned,
  COALESCE(SUM(r.activity), 0)::bigint AS activity,
  COALESCE(SUM(r.balance), 0)::bigint AS balance
FROM groups g
FULL JOIN categories c ON c.group_id = g.id
LEFT JOIN rep.get_category_reports_gate(
  @budget_id::uuid, 
  @month_id::date
) AS r ON r.category_id = c.id
WHERE c.budget_id = @budget_id
GROUP BY g.id, g.name;

-- name: GetMonthGroupReport :one
SELECT 
  @month_id::date AS month,
  g.name::text AS group_name,
  g.id::uuid AS group_id,
  COALESCE(SUM(r.assigned), 0)::bigint AS assigned,
  COALESCE(SUM(r.activity), 0)::bigint AS activity,
  COALESCE(SUM(r.balance), 0)::bigint AS balance
FROM groups g
FULL JOIN categories c ON c.group_id = @group_id::uuid
LEFT JOIN rep.get_category_reports_gate(
  @budget_id::uuid, 
  @month_id::date
) AS r ON r.category_id = c.id
WHERE c.budget_id = @budget_id
GROUP BY g.id, g.name
LIMIT 1;
