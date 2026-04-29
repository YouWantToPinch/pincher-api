-- +goose Up

DROP VIEW IF EXISTS category_reports;

CREATE SCHEMA IF NOT EXISTS rep;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION rep.get_budget_bounds(b_id UUID)
RETURNS TABLE (
  first_month DATE,
  last_month DATE
) AS $$
BEGIN
  RETURN QUERY
  SELECT 
    date_trunc('month', MIN(u.earliest))::date,
    date_trunc('month', MAX(u.latest))::date
  FROM (
    SELECT MIN(a.month) AS earliest, MAX(a.month) AS latest
    FROM assignments a
    JOIN categories c ON a.category_id = c.id
    WHERE c.budget_id = b_id
    
    UNION ALL

    SELECT MIN(t.transaction_date), MAX(t.transaction_date)
    FROM transactions t 
    WHERE t.budget_id = b_id
  ) u;
END;
$$ LANGUAGE plpgsql STABLE;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION rep.get_category_reports(
  b_id UUID,
  start_date DATE,
  end_date DATE
)
RETURNS TABLE (
  month DATE,
  budget_id UUID,
  category_id UUID,
  category_name TEXT,
  assigned BIGINT,
  activity BIGINT,
  balance BIGINT
) AS $$
BEGIN
  RETURN QUERY
  WITH budget_categories AS (
    SELECT id, name
    FROM categories c
    WHERE c.budget_id = b_id
  ),
  totals AS (
    SELECT
      date_trunc('month', agg.dt)::date AS month_id,
      agg.cat_id,
      SUM(agg.val_assigned)::bigint AS val_assigned,
      SUM(agg.val_activity)::bigint AS val_activity
    FROM (
      SELECT a.month AS dt, a.category_id AS cat_id, a.assigned AS val_assigned, 0 AS val_activity
      FROM assignments a
      JOIN categories c ON a.category_id = c.id
      WHERE c.budget_id = b_id
      UNION ALL
      SELECT t.transaction_date, ts.category_id, 0, ts.amount
      FROM transaction_splits ts
      JOIN transactions t ON t.id = ts.transaction_id
      WHERE t.budget_id = b_id
    ) agg
    GROUP BY 1, 2
  ),
  calculated_report AS (
    SELECT
      m.month_id,
      c.id AS cat_id,
      c.name AS cat_name,
      COALESCE(t.val_assigned, 0)::bigint AS assigned,
      COALESCE(t.val_activity, 0)::bigint AS activity,
      (SUM(COALESCE(t.val_assigned, 0) + COALESCE(t.val_activity, 0)) 
          OVER (PARTITION BY c.id ORDER BY m.month_id))::bigint AS balance
    FROM (
      SELECT generate_series(
        (SELECT first_month FROM rep.get_budget_bounds(b_id)),
        date_trunc('month', end_date),
        interval '1 month'
      )::date AS month_id
    ) m
    CROSS JOIN budget_categories c
    LEFT JOIN totals t ON m.month_id = t.month_id AND c.id = t.cat_id
  )
  SELECT 
    cr.month_id::date,
    b_id::uuid,
    cr.cat_id::uuid,
    cr.cat_name::text,
    cr.assigned::bigint,
    cr.activity::bigint,
    cr.balance::bigint
  FROM calculated_report cr
  WHERE cr.month_id >= date_trunc('month', start_date)::date
  ORDER BY cr.month_id, cr.cat_name;
END;
$$ LANGUAGE plpgsql STABLE;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION rep.get_category_reports_gate(
  b_id UUID,
  m_id DATE
)
RETURNS TABLE (
  month DATE,
  budget_id UUID,
  category_id UUID,
  category_name TEXT,
  assigned BIGINT,
  activity BIGINT,
  balance BIGINT
) AS $$
DECLARE
  v_bounds RECORD;
  v_target DATE := date_trunc('month', m_id)::date;
BEGIN
  SELECT * INTO v_bounds FROM rep.get_budget_bounds(b_id);

  IF v_bounds.first_month IS NULL OR v_target < v_bounds.first_month THEN
    RETURN QUERY
    SELECT
      v_target::date,
      b_id::uuid,
      c.id::uuid,
      c.name::text,
      0::bigint,
      0::bigint,
      0::bigint
    FROM categories c
    WHERE c.budget_id = b_id;
    RETURN;
  ELSIF v_target > v_bounds.last_month THEN
    RETURN QUERY 
    SELECT 
      v_target::date,
      b_id::uuid,
      r.category_id::uuid,
      r.category_name::text,
      0::bigint,
      0::bigint,
      r.balance::bigint
    FROM rep.get_category_reports(b_id, v_bounds.first_month, v_bounds.last_month) r
    WHERE r.month = v_bounds.last_month;
    RETURN;
  END IF;

  RETURN QUERY
  SELECT
    r.month::date,
    b_id::uuid,
    r.category_id::uuid,
    r.category_name::text,
    r.assigned::bigint,
    r.activity::bigint,
    r.balance::bigint
  FROM rep.get_category_reports(b_id, v_bounds.first_month, v_bounds.last_month) r
  WHERE r.month = v_target;
END;
$$ LANGUAGE plpgsql STABLE;
-- +goose StatementEnd

-- +goose Down
DROP SCHEMA IF EXISTS rep CASCADE;
