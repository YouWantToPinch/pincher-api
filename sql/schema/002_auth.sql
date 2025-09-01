-- +goose Up
ALTER TABLE users
ADD hashed_password TEXT NOT NULL DEFAULT 'unset';

CREATE TABLE refresh_tokens (
    token CHAR(64) PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE
);

-- +goose Down
ALTER TABLE users
DROP COLUMN hashed_password;
DROP TABLE refresh_tokens;