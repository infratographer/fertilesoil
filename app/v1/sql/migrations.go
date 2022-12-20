// Package sql provides an embedded filesystem containing all the database migrations
package sql

import (
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

// Migrations contain an embedded filesystem with all the sql migration files
//
//go:embed *.sql
var Migrations embed.FS

// BootStrap is a helper function to bootstrap the database
// with the initial schema.
func BootStrap(dialect string, db *sql.DB) error {
	goose.SetBaseFS(Migrations)
	if err := goose.SetDialect(dialect); err != nil {
		return err
	}

	return goose.Up(db, ".")
}
