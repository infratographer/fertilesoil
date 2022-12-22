-- This contains the database model for tracking external
-- directories for a given tree.
-- It's a subset of the directory tree model, and it allows
-- applications to track external directories without having
-- to store the entire directory tree.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE  IF NOT EXISTS tracked_directories (
    id UUID NOT NULL PRIMARY KEY,
    deleted_at TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP DATABASE IF EXISTS tracked_directories;
-- +goose StatementEnd