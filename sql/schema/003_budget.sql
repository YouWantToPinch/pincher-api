-- +goose Up
CREATE TABLE budgets (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    admin_id UUID NOT NULL,
    name VARCHAR(30) NOT NULL,
    notes TEXT,
    FOREIGN KEY (admin_id) REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE TABLE budgets_users (
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    budget_id UUID NOT NULL,
    user_id UUID NOT NULL,
    member_role VARCHAR(12) NOT NULL,
    FOREIGN KEY (budget_id) REFERENCES budgets(id)
        ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE,
    UNIQUE(budget_id, user_id)
);

-- +goose Down
DROP TABLE budgets_users;
DROP TABLE budgets;