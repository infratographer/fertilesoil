//go:build testtools
// +build testtools

package utils

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/pressly/goose/v3"

	"github.com/infratographer/fertilesoil/storage/crdb/migrations"
)

type StopServerFunc func()

var gooseMutex sync.Mutex

func NewTestDBServer() (*url.URL, StopServerFunc, error) {
	srv, err := testserver.NewTestServer()
	if err != nil {
		return nil, nil, err
	}

	if err := srv.WaitForInit(); err != nil {
		return nil, nil, err
	}

	dbURL := srv.PGURL()

	// Reset Path so we can use the database in general
	dbURL.Path = "/"

	return dbURL, func() {
		srv.Stop()
	}, nil
}

func NewTestDBServerOrDie() (*url.URL, StopServerFunc) {
	dbURL, cleanup, err := NewTestDBServer()
	if err != nil {
		panic(fmt.Sprintf("error creating test database server: %v", err))
	}
	return dbURL, cleanup
}

// Returns a new test database for an application.
// The database is not migrated.
func GetNewTestDBForApp(t *testing.T, baseDBURL *url.URL) *sql.DB {
	t.Helper()

	return getNewTestDB(t, baseDBURL, "app")
}

// Returns a new test database for the tree manager
// with the migrations applied.
func GetNewTestDB(t *testing.T, baseDBURL *url.URL) *sql.DB {
	t.Helper()

	gooseMutex.Lock()
	defer gooseMutex.Unlock()

	dbConn := getNewTestDB(t, baseDBURL, "")

	goose.SetBaseFS(migrations.Migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("error setting dialect: %v", err)
	}

	if err := goose.Up(dbConn, "."); err != nil {
		t.Fatalf("error running migrations: %v", err)
	}
	return dbConn
}

func getNewTestDB(t *testing.T, baseDBURL *url.URL, suffix string) *sql.DB {
	t.Helper()

	dbName := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_"))
	dbName += suffix

	baseDB, err := sql.Open("postgres", baseDBURL.String())
	if err != nil {
		t.Fatalf("error opening database: %v", err)
	}

	if _, err := baseDB.Exec("CREATE DATABASE " + dbName); err != nil {
		t.Fatalf("error creating database: %v", err)
	}

	dbURL := baseDBURL.JoinPath(dbName)
	dbConn, err := sql.Open("postgres", dbURL.String())
	if err != nil {
		t.Fatalf("error opening database: %v", err)
	}

	return dbConn
}
