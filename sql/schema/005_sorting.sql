-- +goose Up
CREATE TABLE groups (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    user_id UUID NOT NULL,
    name TEXT NOT NULL,
    notes TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE TABLE categories (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    user_id UUID NOT NULL,
    name TEXT NOT NULL,
    group_id UUID,
    notes TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE,
    FOREIGN KEY (group_id) REFERENCES groups(id)
);

CREATE TABLE transaction_categories (
  transaction_id UUID,
  category_id UUID,
  PRIMARY KEY (transaction_id, category_id),
  FOREIGN KEY (transaction_id) REFERENCES transactions(id),
  FOREIGN KEY (category_id) REFERENCES categories(id)
);

-- +goose Down
DROP TABLE transaction_categories;
DROP TABLE categories;
DROP TABLE groups;