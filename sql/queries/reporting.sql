-- name: GetMonthReport :one
WITH reports AS (
    SELECT DISTINCT ON (category_id)
        month,
        activity,
        assigned,
        balance
    FROM category_reports
    WHERE budget_id = @budget_id::uuid 
      AND month <= date_trunc('month', @month_id::date)
    ORDER BY category_id, month DESC
)
SELECT 
    COALESCE(SUM(assigned) FILTER (WHERE month = date_trunc('month', @month_id::date)), 0)::bigint AS assigned,
    COALESCE(SUM(activity) FILTER (WHERE month = date_trunc('month', @month_id::date)), 0)::bigint AS activity,
    COALESCE(SUM(balance), 0)::bigint AS balance
FROM reports;

-- name: GetMonthCategoryReports :many
WITH target AS (
    SELECT date_trunc('month', @month_id::date) AS month
)
SELECT DISTINCT ON (c.id)
    c.id AS category_id,
    c.name AS category_name,
    c.budget_id,
    COALESCE(cr.month, t.month) AS month, -- Use target month if View row is missing
    CASE WHEN cr.month = t.month THEN cr.activity ELSE 0 END AS activity,
    CASE WHEN cr.month = t.month THEN cr.assigned ELSE 0 END AS assigned,
    COALESCE(cr.balance, 0) AS balance
FROM categories c
CROSS JOIN target t
LEFT JOIN category_reports cr 
    ON c.id = cr.category_id 
    AND cr.month <= t.month
WHERE c.budget_id = @budget_id::uuid
ORDER BY c.id, cr.month DESC;

-- name: GetMonthCategoryReport :one
SELECT 
    c.id AS category_id,
    c.name AS category_name,
    c.budget_id,
    COALESCE(cr.month, t.month) AS month,
    CASE WHEN cr.month = t.month THEN cr.activity ELSE 0 END AS activity,
    CASE WHEN cr.month = t.month THEN cr.assigned ELSE 0 END AS assigned,
    COALESCE(cr.balance, 0) AS balance
FROM categories c
CROSS JOIN (SELECT date_trunc('month', @month_id::date) AS month) t
LEFT JOIN category_reports cr 
    ON c.id = cr.category_id 
    AND cr.month <= t.month
WHERE c.budget_id = @budget_id::uuid AND c.id = @category_id::uuid
ORDER BY cr.month DESC
LIMIT 1;
