-- +goose Up
CREATE TABLE budgets (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    name VARCHAR(12) NOT NULL,
    notes TEXT
);

CREATE TABLE budgets_users (
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    budget_id UUID NOT NULL,
    user_id UUID NOT NULL,
    user_role VARCHAR(21) NOT NULL,
    FOREIGN KEY (budget_id) REFERENCES budgets(id)
        ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE,
    UNIQUE(budget_id, user_id)
);

-- +goose Down
DROP TABLE budgets_users;
DROP TABLE budgets;