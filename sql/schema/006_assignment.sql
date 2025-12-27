-- +goose Up

CREATE TABLE assignments (
  month DATE NOT NULL,
  category_id UUID NOT NULL,
  assigned BIGINT NOT NULL,
  UNIQUE (month, category_id),
  FOREIGN KEY(category_id) REFERENCES categories(id)
    ON DELETE CASCADE
);

CREATE VIEW category_reports AS
WITH month_range AS (
    SELECT 
        date_trunc('month', MIN(months)) AS first_month,
        date_trunc('month', MAX(months)) AS last_month
    FROM (
        SELECT a.month AS months FROM assignments a
        UNION ALL
        SELECT t.transaction_date AS months FROM transactions t
    ) AS all_months
),
all_months AS (
    SELECT generate_series(first_month, last_month, interval '1 month')::date AS month
    FROM month_range
),
report_identifiers AS (
    SELECT m.month, c.id AS category_id, c.name AS category_name
    FROM all_months m
    CROSS JOIN categories c
),
agg_assignments AS (
    SELECT date_trunc('month', month)::date AS month, category_id, SUM(assigned)::bigint AS assigned
    FROM assignments
    GROUP BY 1, 2
),
agg_activity AS (
    SELECT date_trunc('month', t.transaction_date)::date AS month, ts.category_id, SUM(ts.amount)::bigint AS activity
    FROM transaction_splits ts
    JOIN transactions t ON t.id = ts.transaction_id
    GROUP BY 1, 2
),
report AS (
    SELECT
        rep.month,
        rep.category_id,
        rep.category_name,
        COALESCE(aa.assigned, 0) AS assigned,
        COALESCE(ta.activity, 0) AS activity
    FROM report_identifiers rep
    LEFT JOIN agg_assignments aa
      ON aa.category_id = rep.category_id AND aa.month = rep.month
    LEFT JOIN agg_activity ta
      ON ta.category_id = rep.category_id AND ta.month = rep.month
)
SELECT
    month,
    category_name,
    category_id,
    assigned,
    activity,
    (SUM(assigned + activity) OVER (PARTITION BY category_id ORDER BY month))::bigint AS balance
FROM report
ORDER BY month, category_name;


-- +goose Down
DROP VIEW category_reports;
DROP TABLE assignments;
