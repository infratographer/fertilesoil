// Package db provides an embedded filesystem containing all the database migrations
package migrations

import (
	"embed"
)

// Migrations contain an embedded filesystem with all the sql migration files
//
//go:embed *.sql
var Migrations embed.FS
