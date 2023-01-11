-- This contains the database model for the directory tree.
-- Note that currently this is tied to CockroachDB.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE  IF NOT EXISTS directories (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    name TEXT NOT NULL,
    metadata JSONB NOT NULL,
    parent_id UUID REFERENCES directories(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
    CONSTRAINT parent_child_not_equal CHECK (id != parent_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP DATABASE IF EXISTS directories;
-- +goose StatementEnd
