-- +goose Up
CREATE TABLE groups (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    budget_id UUID NOT NULL,
    name VARCHAR(50) NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    UNIQUE(budget_id, name),
    FOREIGN KEY (budget_id) REFERENCES budgets(id)
        ON DELETE CASCADE
);

CREATE TABLE categories (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    budget_id UUID NOT NULL,
    name VARCHAR(50) NOT NULL,
    group_id UUID,
    notes TEXT NOT NULL DEFAULT '',
    UNIQUE(budget_id, name),
    FOREIGN KEY (budget_id) REFERENCES budgets(id)
        ON DELETE CASCADE,
    FOREIGN KEY (group_id) REFERENCES groups(id)
);

-- +goose Down
DROP TABLE categories;
DROP TABLE groups;
