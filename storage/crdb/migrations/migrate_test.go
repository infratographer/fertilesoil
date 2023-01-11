package migrations_test

import (
	"database/sql"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"

	"github.com/infratographer/fertilesoil/storage/crdb/migrations"
)

func TestMigrations(t *testing.T) {
	t.Parallel()

	ts, crdberr := testserver.NewTestServer()
	assert.NoError(t, crdberr)
	defer ts.Stop()

	dbdialect := "postgres"
	dbConn, dbopenerr := sql.Open(dbdialect, ts.PGURL().String())
	assert.NoError(t, dbopenerr, "failed to open db connection")

	goose.SetBaseFS(migrations.Migrations)
	assert.NoError(t, goose.SetDialect(dbdialect), "failed to set dialect")
	assert.NoError(t, goose.Up(dbConn, "."), "failed to run migrations")
	assert.NoError(t, goose.Down(dbConn, "."), "failed to run migrations")
}

func TestMigrate(t *testing.T) {
	t.Parallel()

	ts, crdberr := testserver.NewTestServer()
	assert.NoError(t, crdberr)
	defer ts.Stop()

	dbdialect := "postgres"
	dbConn, dbopenerr := sql.Open(dbdialect, ts.PGURL().String())
	assert.NoError(t, dbopenerr, "failed to open db connection")

	assert.NoError(t, migrations.Migrate(dbConn), "failed to set dialect")
}
