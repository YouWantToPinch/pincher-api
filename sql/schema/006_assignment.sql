-- +goose Up

CREATE TABLE assignments (
  month DATE NOT NULL,
  category_id UUID NOT NULL,
  assigned BIGINT NOT NULL,
  UNIQUE (month, category_id),
  FOREIGN KEY(category_id) REFERENCES categories(id)
);

CREATE VIEW month_report AS 
SELECT
  a.month,
  c.name,
  a.assigned,
  a.category_id,
  SUM(ts.amount)::bigint AS activity,
  (SUM(ts.amount) + SUM(a.assigned))::bigint AS balance
FROM assignments a
JOIN categories c ON c.id = a.category_id
JOIN transaction_splits ts ON ts.category_id = c.id
JOIN transactions t ON t.id = ts.transaction_id
  AND EXTRACT(MONTH FROM t.transaction_date) = EXTRACT(MONTH FROM a.month)
  AND EXTRACT(YEAR FROM t.transaction_date) = EXTRACT(YEAR FROM a.month)
GROUP BY
  a.month,
  c.name,
  a.category_id,
  a.assigned;

-- +goose Down
DROP VIEW month_report;
DROP TABLE assignments;
