-- +goose Up

CREATE TABLE assignments (
  month DATE NOT NULL,
  category_id UUID NOT NULL,
  assigned BIGINT NOT NULL,
  UNIQUE (month, category_id),
  FOREIGN KEY(category_id) REFERENCES categories(id)
    ON DELETE CASCADE
);

CREATE VIEW month_report AS 
SELECT
  a.month,
  c.name AS category_name,
  a.category_id,
  a.assigned,

  -- activity: same month
  SUM(
    CASE 
      WHEN date_part('year', t.transaction_date) = date_part('year', a.month)
       AND date_part('month', t.transaction_date) = date_part('month', a.month)
      THEN ts.amount
      ELSE 0
    END
  )::bigint AS activity,

  -- balance: up to and including that month
  COALESCE(SUM(
    CASE 
      WHEN (date_part('year', t.transaction_date) < date_part('year', a.month))
        OR (
          date_part('year', t.transaction_date) = date_part('year', a.month)
          AND date_part('month', t.transaction_date) <= date_part('month', a.month)
        )
      THEN ts.amount
      ELSE 0
    END
  )::bigint + SUM(a.assigned), 0)::bigint AS balance

FROM assignments a
JOIN categories c ON c.id = a.category_id
JOIN transaction_splits ts ON ts.category_id = c.id
JOIN transactions t ON t.id = ts.transaction_id
GROUP BY
  a.month,
  c.name,
  a.category_id,
  a.assigned;

-- +goose Down
DROP VIEW month_report;
DROP TABLE assignments;
