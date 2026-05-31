-- +goose Up
DROP VIEW IF EXISTS transaction_details;

CREATE VIEW transaction_details AS
SELECT 
  t.id,
  t.created_at,
  t.updated_at,
  t.transaction_date,
  t.transaction_type,
  t.notes,
  COALESCE(
    p.name,
    'Transfer'
  ) AS payee_name,
  b.name AS budget_name,
  a.name AS account_name,
  u.username AS logger_name,
  SUM(ts.amount)::bigint AS total_amount,
  jsonb_object_agg(COALESCE(c.name, 'Uncategorized'), ts.amount) AS splits,
  t.cleared
FROM transactions t
JOIN transaction_splits ts ON t.id = ts.transaction_id
LEFT JOIN categories c ON ts.category_id = c.id
LEFT JOIN payees p ON t.payee_id = p.id
LEFT JOIN accounts a ON t.account_id = a.id
LEFT JOIN users u ON t.logger_id = u.id
LEFT JOIN budgets b ON t.budget_id = b.id
GROUP BY
    t.id,
    t.created_at,
    t.updated_at,
    t.transaction_date,
    t.transaction_type,
    t.notes,
    p.name,
    a.name,
    u.username,
    b.name
ORDER BY
    t.transaction_date DESC;


-- +goose Down
DROP VIEW transaction_details;
