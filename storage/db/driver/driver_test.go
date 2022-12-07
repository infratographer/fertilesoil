package driver_test

import (
	"context"
	"database/sql"
	"net/url"
	"strings"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"

	v1 "github.com/JAORMX/fertilesoil/api/v1"
	"github.com/JAORMX/fertilesoil/storage"
	"github.com/JAORMX/fertilesoil/storage/db/driver"
	"github.com/JAORMX/fertilesoil/storage/db/migrations"
)

var (
	baseDBURL *url.URL
)

func TestMain(m *testing.M) {
	// Spawn CRDB instance
	ts, err := testserver.NewTestServer()
	if err != nil {
		panic(err)
	}
	defer ts.Stop()

	ts.WaitForInit()

	baseDBURL = ts.PGURL()
	// Reset Path so we can use the database in general
	baseDBURL.Path = "/"

	m.Run()
}

func getNewDB(t *testing.T) *sql.DB {
	dbName := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_"))

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

	goose.SetBaseFS(migrations.Migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("error setting dialect: %v", err)
	}

	if err := goose.Up(dbConn, "migrations"); err != nil {
		t.Fatalf("error running migrations: %v", err)
	}
	return dbConn
}

func withRootDir(t *testing.T, store storage.Storage) *v1.Directory {
	d := &v1.Directory{
		Name: "root",
	}

	rd, err := store.CreateRoot(context.Background(), d)
	assert.NoError(t, err, "error creating root directory")
	assert.NotNil(t, rd.Metadata, "metadata should not be nil")
	assert.Equal(t, d.Name, rd.Name, "name should match")

	return d
}

func TestCreateAndGetRoot(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	rd := withRootDir(t, store)

	// retrieve from db
	queryrootdir, qerr := store.GetDirectory(context.Background(), rd.ID)
	assert.NoError(t, qerr, "error querying db")
	assert.Equal(t, rd.ID, queryrootdir.ID, "id should match")
	assert.Equal(t, rd.Name, queryrootdir.Name, "name should match")
}

func TestListRootOneRoot(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	rd := withRootDir(t, store)

	r, err := store.ListRoots(context.Background())
	assert.NoError(t, err, "should not have errored")
	assert.Len(t, r, 1, "should have 1 root")
	assert.Contains(t, r, rd.ID, "should contain root")
}

func TestListMultipleRoots(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	rd1 := withRootDir(t, store)
	rd2 := withRootDir(t, store)
	rd3 := withRootDir(t, store)
	rd4 := withRootDir(t, store)

	r, err := store.ListRoots(context.Background())
	assert.NoError(t, err, "should not have errored")
	assert.Len(t, r, 4, "should have 4 roots")
	assert.Contains(t, r, rd1.ID, "should contain root")
	assert.Contains(t, r, rd2.ID, "should contain root")
	assert.Contains(t, r, rd3.ID, "should contain root")
	assert.Contains(t, r, rd4.ID, "should contain root")
}

func TestCreateMultipleRoots(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	rd1 := withRootDir(t, store)
	rd2 := withRootDir(t, store)

	assert.NotEqual(t, rd1.ID, rd2.ID, "ids should not match")

	// retrieve from db
	rows, err := db.Query("SELECT * FROM directories WHERE parent_id IS NULL")
	assert.NoError(t, err, "error querying db")

	// We should have 2 rows
	var count int
	for rows.Next() {
		count++
	}
	assert.Equal(t, 2, count, "should have 2 rows")
}

func TestCantCreateRootWithParent(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	d := &v1.Directory{
		Name:   "root",
		Parent: &v1.Directory{},
	}

	rd, err := store.CreateRoot(context.Background(), d)
	assert.Error(t, err, "should have errored")
	assert.Nil(t, rd, "should be nil")
}

func TestCreateAndGetDirectory(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	rootdir := withRootDir(t, store)

	d := &v1.Directory{
		Name:   "testdir",
		Parent: rootdir,
	}

	rd, err := store.CreateDirectory(context.Background(), d)
	assert.NoError(t, err, "error creating directory")
	assert.NotNil(t, rd.Metadata, "metadata should not be nil")
	assert.Equal(t, d.Name, rd.Name, "name should match")
	assert.Equal(t, d.Parent.ID, rd.Parent.ID, "parent id should match")

	// retrieve from db
	querydir, qerr := store.GetDirectory(context.Background(), rd.ID)
	assert.NoError(t, qerr, "error querying db")
	assert.Equal(t, rd.ID, querydir.ID, "id should match")

	// should have only 1 directory that's not root
	rows, err := db.Query("SELECT * FROM directories WHERE parent_id IS NOT NULL")
	assert.NoError(t, err, "error querying db")

	// We should have 2 rows
	var count int
	for rows.Next() {
		count++
	}
	assert.Equal(t, 1, count, "should have 1 rows")
}

func TestCreateDirectoryWithParentThatDoesntExist(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	d := &v1.Directory{
		Name:   "testdir",
		Parent: &v1.Directory{ID: v1.DirectoryID(uuid.New())},
	}

	rd, err := store.CreateDirectory(context.Background(), d)
	assert.Error(t, err, "should have errored")
	assert.Nil(t, rd, "should be nil")
}

func TestCreateDirectoryWithoutParent(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	d := &v1.Directory{
		Name: "testdir",
	}

	rd, err := store.CreateDirectory(context.Background(), d)
	assert.Error(t, err, "should have errored")
	assert.Nil(t, rd, "should be nil")
}

func TestQueryUnknownDirectory(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	d, err := store.GetDirectory(context.Background(), v1.DirectoryID(uuid.New()))
	assert.ErrorIs(t, err, driver.ErrDirectoryNotFound, "should have errored")
	assert.Nil(t, d, "should be nil")
}

func TestGetSingleParent(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	rootdir := withRootDir(t, store)

	d := &v1.Directory{
		Name:   "testdir",
		Parent: rootdir,
	}

	rd, err := store.CreateDirectory(context.Background(), d)
	assert.NoError(t, err, "error creating directory")

	parents, gperr := store.GetParents(context.Background(), rd.ID)
	assert.NoError(t, gperr, "error getting parents")
	assert.Len(t, parents, 1, "should have 1 parent")

	assert.Equal(t, rootdir.ID, parents[0], "id should match")
}

func TestGetMultipleParents(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	rootdir := withRootDir(t, store)

	d1 := &v1.Directory{
		Name:   "testdir1",
		Parent: rootdir,
	}
	d2 := &v1.Directory{
		Name:   "testdir2",
		Parent: d1,
	}
	d3 := &v1.Directory{
		Name:   "testdir3",
		Parent: d2,
	}

	_, err := store.CreateDirectory(context.Background(), d1)
	assert.NoError(t, err, "error creating directory")

	_, err = store.CreateDirectory(context.Background(), d2)
	assert.NoError(t, err, "error creating directory")

	rd3, err := store.CreateDirectory(context.Background(), d3)
	assert.NoError(t, err, "error creating directory")

	parents, gperr := store.GetParents(context.Background(), rd3.ID)
	assert.NoError(t, gperr, "error getting parents")
	assert.Len(t, parents, 3, "should have 3 parents")

	assert.Equal(t, d2.ID, parents[0], "id should match")
	assert.Equal(t, d1.ID, parents[1], "id should match")
	assert.Equal(t, rootdir.ID, parents[2], "id should match")
}

func TestGetParentFromRootDirShouldReturnEmpty(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	rootdir := withRootDir(t, store)

	parents, gperr := store.GetParents(context.Background(), rootdir.ID)
	assert.NoError(t, gperr, "error getting parents")
	assert.Len(t, parents, 0, "should have 0 parents")
}

func TestGetParentsFromUnknownShouldFail(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	parents, err := store.GetParents(context.Background(), v1.DirectoryID(uuid.New()))
	assert.ErrorIs(t, err, driver.ErrDirectoryNotFound, "should have errored")
	assert.Nil(t, parents, "should be nil")
}

func TestGetChildrenBasic(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	rootdir := withRootDir(t, store)

	d := &v1.Directory{
		Name:   "testdir",
		Parent: rootdir,
	}

	rd, err := store.CreateDirectory(context.Background(), d)
	assert.NoError(t, err, "error creating directory")

	children, gperr := store.GetChildren(context.Background(), rootdir.ID)
	assert.NoError(t, gperr, "error getting children")
	assert.Len(t, children, 1, "should have 1 child")

	assert.Equal(t, rd.ID, children[0], "id should match")
}

func TestGetChildrenMultiple(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	rootdir := withRootDir(t, store)

	d1 := &v1.Directory{
		Name:   "testdir1",
		Parent: rootdir,
	}

	d2 := &v1.Directory{
		Name:   "testdir2",
		Parent: rootdir,
	}

	d3 := &v1.Directory{
		Name:   "testdir3",
		Parent: rootdir,
	}

	d4 := &v1.Directory{
		Name:   "testdir4",
		Parent: d1,
	}

	rd1, err := store.CreateDirectory(context.Background(), d1)
	assert.NoError(t, err, "error creating directory")

	rd2, err := store.CreateDirectory(context.Background(), d2)
	assert.NoError(t, err, "error creating directory")

	rd3, err := store.CreateDirectory(context.Background(), d3)
	assert.NoError(t, err, "error creating directory")

	rd4, err := store.CreateDirectory(context.Background(), d4)
	assert.NoError(t, err, "error creating directory")

	children, gperr := store.GetChildren(context.Background(), rootdir.ID)
	assert.NoError(t, gperr, "error getting children")

	assert.Len(t, children, 4, "should have 4 children")

	assert.Contains(t, children, rd1.ID, "should contain id")
	assert.Contains(t, children, rd2.ID, "should contain id")
	assert.Contains(t, children, rd3.ID, "should contain id")
	assert.Contains(t, children, rd4.ID, "should contain id")
}

func TestGetChildrenMayReturnEmptyAppropriately(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	rootdir := withRootDir(t, store)

	children, err := store.GetChildren(context.Background(), rootdir.ID)
	assert.NoError(t, err, "should have errored")
	assert.Len(t, children, 0, "should have 0 children")
}

func TestGetChildrenFromUnknownReturnsEmpty(t *testing.T) {
	db := getNewDB(t)
	store := driver.NewDBDriver(db)

	children, err := store.GetChildren(context.Background(), v1.DirectoryID(uuid.New()))
	assert.NoError(t, err, "should have errored")
	assert.Len(t, children, 0, "should have 0 children")
}
