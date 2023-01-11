// Package db provides an embedded filesystem containing all the database migrations
package migrations

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"

	appsqlmig "github.com/infratographer/fertilesoil/app/v1/sql/migrations"
)

const (
	dialect = "postgres"
)

// Migrations contain an embedded filesystem with all the sql migration files
//
//go:embed *.sql
var Migrations embed.FS

// Migrate runs all the migrations in the migrations directory.
// Note that goose is not thread-safe, and so, this function should
// not be called concurrently.
func Migrate(db *sql.DB) error {
	if err := goose.SetDialect(dialect); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// This ensures that we have the latest version of the app migrations
	// in the database. This is where we get the tracked_directories table
	// and the app migrations are added to it.
	if err := appsqlmig.BootStrap(dialect, db); err != nil {
		return fmt.Errorf("failed to bootstrap app migrations: %w", err)
	}

	goose.SetBaseFS(Migrations)

	return goose.Up(db, ".")
}
