-- name: GetMonthReport :one
SELECT 
  month::DATE AS month,
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
  month::DATE AS month,
  category_id::uuid AS category_id,
  category_name::text AS category_name,
  COALESCE(assigned, 0)::bigint AS assigned,
  COALESCE(activity, 0)::bigint AS activity,
  COALESCE(balance, 0)::bigint AS balance
FROM rep.get_category_reports_gate(
  @budget_id::uuid, 
  @month_id::date
)
ORDER BY category_name;

-- name: GetMonthCategoryReport :one
SELECT 
  month::DATE AS month,
  category_id::uuid AS category_id,
  category_name::text AS category_name,
  COALESCE(assigned, 0)::bigint AS assigned,
  COALESCE(activity, 0)::bigint AS activity,
  COALESCE(balance, 0)::bigint AS balance
FROM rep.get_category_reports_gate(
  @budget_id::uuid, 
  @month_id::date
)
WHERE category_id = @category_id::uuid
LIMIT 1;
