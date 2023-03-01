package driver_test

import (
	"context"
	"database/sql"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	v1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/storage"
	"github.com/infratographer/fertilesoil/storage/crdb/driver"
	"github.com/infratographer/fertilesoil/storage/crdb/utils"
)

func TestCreateAndGetRoot(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rd := withRootDir(t, store)

	// retrieve from db
	queryrootdir, qerr := store.GetDirectory(context.Background(), rd.Id)
	assert.NoError(t, qerr, "error querying db")
	assert.Equal(t, rd.Id, queryrootdir.Id, "id should match")
	assert.Equal(t, rd.Name, queryrootdir.Name, "name should match")
}

func TestListRootOneRoot(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rd := withRootDir(t, store)

	r, err := store.ListRoots(context.Background())
	assert.NoError(t, err, "should not have errored")
	assert.Len(t, r, 1, "should have 1 root")
	assert.Contains(t, r, rd.Id, "should contain root")

	r, err = store.ListRoots(context.Background(), storage.WithDeletedDirectories)
	assert.NoError(t, err, "should not have errored")
	assert.Len(t, r, 1, "should have 1 root")
	assert.Contains(t, r, rd.Id, "should contain root")
}

func TestListMultipleRoots(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rd1 := withRootDir(t, store)
	rd2 := withRootDir(t, store)
	rd3 := withRootDir(t, store)
	rd4 := withRootDir(t, store)

	r, err := store.ListRoots(context.Background())
	assert.NoError(t, err, "should not have errored")
	assert.Len(t, r, 4, "should have 4 roots")
	assert.Contains(t, r, rd1.Id, "should contain root")
	assert.Contains(t, r, rd2.Id, "should contain root")
	assert.Contains(t, r, rd3.Id, "should contain root")
	assert.Contains(t, r, rd4.Id, "should contain root")
}

func TestCreateMultipleRoots(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rd1 := withRootDir(t, store)
	rd2 := withRootDir(t, store)

	assert.NotEqual(t, rd1.Id, rd2.Id, "ids should not match")

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
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	randomPID := v1.DirectoryID(uuid.New())
	d := &v1.Directory{
		Name:   "root",
		Parent: &randomPID,
	}

	rd, err := store.CreateRoot(context.Background(), d)
	assert.Error(t, err, "should have errored")
	assert.Nil(t, rd, "should be nil")
}

func TestCreateAndGetDirectory(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rootdir := withRootDir(t, store)

	d := &v1.Directory{
		Name:   "testdir",
		Parent: &rootdir.Id,
	}

	rd, err := store.CreateDirectory(context.Background(), d)
	assert.NoError(t, err, "error creating directory")
	assert.NotNil(t, rd.Metadata, "metadata should not be nil")
	assert.Equal(t, d.Name, rd.Name, "name should match")
	assert.Equal(t, d.Parent, rd.Parent, "parent id should match")

	// retrieve from db
	querydir, qerr := store.GetDirectory(context.Background(), rd.Id)
	assert.NoError(t, qerr, "error querying db")
	assert.Equal(t, rd.Id, querydir.Id, "id should match")

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

// createDirectoryHierarchy will create the specified depth of directories starting from a new root directory.
func createDirectoryHierarchy(store storage.DirectoryAdmin, depth int) (root, last *v1.Directory, err error) {
	for i := 0; i < depth; i++ {
		d := &v1.Directory{
			Name: "dir" + strconv.Itoa(i),
		}
		if last != nil {
			d.Parent = &last.Id
			last, err = store.CreateDirectory(context.Background(), d)
		} else {
			last, err = store.CreateRoot(context.Background(), d)
		}

		if err != nil {
			return nil, nil, err
		}

		if root == nil {
			root = last
		}
	}

	return root, last, nil
}

func TestDirectoryPagination(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rootdir, lastDir, err := createDirectoryHierarchy(store, 17)
	assert.NoError(t, err, "creating testing directory hierarchy should not return an error")

	// Creating large amount of root directories
	for i := 1; i < 17; i++ {
		d := &v1.Directory{
			Name: "root" + strconv.Itoa(i),
		}
		_, err := store.CreateRoot(context.Background(), d)
		assert.NoError(t, err, "error creating root directory")
	}

	// Listing Root directories

	// Without pagination defined, should return first page with default page size
	results, err := store.ListRoots(context.Background())
	assert.NoError(t, err, "listing roots should not return error")

	assert.Len(t, results, storage.DefaultPageSize, "unexpected default page size directories returned")

	// First page with default page size should match without
	page1, err := store.ListRoots(context.Background(), storage.Pagination(1, 0))
	assert.NoError(t, err, "listing roots should not return error")

	assert.Len(t, page1, storage.DefaultPageSize, "unexpected default page size directories returned")
	assert.Equal(t, results, page1, "page 1 doesn't match default page results")

	// Second page with default page size should be a partial return
	page2, err := store.ListRoots(context.Background(), storage.Pagination(2, 0))
	assert.NoError(t, err, "listing roots should not return error")

	assert.Len(t, page2, 7, "unexpected number of results for the second page")

	// Third page with default page size should be empty
	results, err = store.ListRoots(context.Background(), storage.Pagination(3, 0))
	assert.NoError(t, err, "listing roots should not return error")

	assert.Len(t, results, 0, "unexpected number of results for the third page")

	// Larger limit
	results, err = store.ListRoots(context.Background(), storage.Pagination(1, 20))
	assert.NoError(t, err, "listing roots should not return error")

	assert.Len(t, results, 17, "unexpected number of results with larger limit")

	// Ensure page2 does not have any duplicate ids from page 1
	for _, did := range page1 {
		assert.NotContainsf(t, page2, did, "page 2 should not contain id %s from page 1", did)
	}

	// Getting Children

	// Without pagination defined, should return first page with default page size
	results, err = store.GetChildren(context.Background(), rootdir.Id)
	assert.NoError(t, err, "listing children should not return error")

	assert.Len(t, results, storage.DefaultPageSize, "unexpected default page size directories returned")

	// First page with default page size should match without
	page1, err = store.GetChildren(context.Background(), rootdir.Id, storage.Pagination(1, 0))
	assert.NoError(t, err, "listing children should not return error")

	assert.Len(t, page1, storage.DefaultPageSize, "unexpected first page size directories returned")
	assert.Equal(t, results, page1, "page 1 doesn't match default page results")

	// Second page with default page size should be a partial return
	page2, err = store.GetChildren(context.Background(), rootdir.Id, storage.Pagination(2, 0))
	assert.NoError(t, err, "listing children should not return error")

	assert.Len(t, page2, 6, "unexpected number of results for the second page")

	// Third page with default page size should be empty
	results, err = store.GetChildren(context.Background(), rootdir.Id, storage.Pagination(3, 0))
	assert.NoError(t, err, "listing children should not return error")

	assert.Len(t, results, 0, "unexpected number of results for the third page")

	// Larger limit
	results, err = store.GetChildren(context.Background(), rootdir.Id, storage.Pagination(1, 20))
	assert.NoError(t, err, "listing children should not return error")

	assert.Len(t, results, 16, "unexpected number of results with larger limit")

	// Ensure page2 does not have any duplicate ids from page 1
	for _, did := range page1 {
		assert.NotContainsf(t, page2, did, "page 2 should not contain id %s from page 1", did)
	}

	// Getting Parents

	// Without pagination defined, should return first page with default page size
	results, err = store.GetParents(context.Background(), lastDir.Id)
	assert.NoError(t, err, "listing parents should not return error")

	assert.Len(t, results, storage.DefaultPageSize, "unexpected default page size directories returned")

	// First page with default page size should match without
	page1, err = store.GetParents(context.Background(), lastDir.Id, storage.Pagination(1, 0))
	assert.NoError(t, err, "listing parents should not return error")

	assert.Len(t, page1, storage.DefaultPageSize, "unexpected first page size directories returned")
	assert.Equal(t, results, page1, "page 1 doesn't match default page results")

	// Second page with default page size should be a partial return
	page2, err = store.GetParents(context.Background(), lastDir.Id, storage.Pagination(2, 0))
	assert.NoError(t, err, "listing parents should not return error")

	assert.Len(t, page2, 6, "unexpected number of results for the second page")

	// Third page with default page size should be empty
	results, err = store.GetParents(context.Background(), lastDir.Id, storage.Pagination(3, 0))
	assert.NoError(t, err, "listing parents should not return error")

	assert.Len(t, results, 0, "unexpected number of results for the third page")

	// Larger limit
	results, err = store.GetParents(context.Background(), lastDir.Id, storage.Pagination(1, 20))
	assert.NoError(t, err, "listing parents should not return error")

	assert.Len(t, results, 16, "unexpected number of results with larger limit")

	// Ensure page2 does not have any duplicate ids from page 1
	for _, did := range page1 {
		assert.NotContainsf(t, page2, did, "page 2 should not contain id %s from page 1", did)
	}

	// Getting Parents Until

	// Without pagination defined, should return first page with default page size
	results, err = store.GetParentsUntilAncestor(context.Background(), lastDir.Id, rootdir.Id)
	assert.NoError(t, err, "listing parents until ancestor should not return error")

	assert.Len(t, results, storage.DefaultPageSize, "unexpected default page size directories returned")

	// First page with default page size should match without
	page1, err = store.GetParentsUntilAncestor(context.Background(), lastDir.Id, rootdir.Id, storage.Pagination(1, 0))
	assert.NoError(t, err, "listing parents until ancestor should not return error")

	assert.Len(t, page1, storage.DefaultPageSize, "unexpected first page size directories returned")
	assert.Equal(t, results, page1, "page 1 doesn't match default page results")

	// Second page with default page size should be a partial return
	page2, err = store.GetParentsUntilAncestor(context.Background(), lastDir.Id, rootdir.Id, storage.Pagination(2, 0))
	assert.NoError(t, err, "listing parents until ancestor should not return error")

	assert.Len(t, page2, 6, "unexpected number of results for the second page")

	// Third page with default page size should be empty
	results, err = store.GetParentsUntilAncestor(context.Background(), lastDir.Id, rootdir.Id, storage.Pagination(3, 0))
	assert.NoError(t, err, "listing parents until ancestor should not return error")

	assert.Len(t, results, 0, "unexpected number of results for the third page")

	// Larger limit
	results, err = store.GetParentsUntilAncestor(context.Background(), lastDir.Id, rootdir.Id, storage.Pagination(1, 20))
	assert.NoError(t, err, "listing parents until ancestor should not return error")

	assert.Len(t, results, 16, "unexpected number of results with larger limit")

	// Ensure page2 does not have any duplicate ids from page 1
	for _, did := range page1 {
		assert.NotContainsf(t, page2, did, "page 2 should not contain id %s from page 1", did)
	}
}

func TestCreateAndUpdateDirectory(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rootdir := withRootDir(t, store)

	d := &v1.Directory{
		Name:   "testdir",
		Parent: &rootdir.Id,
	}

	rd, err := store.CreateDirectory(context.Background(), d)
	assert.NoError(t, err, "error creating directory")
	assert.NotNil(t, rd.Metadata, "metadata should not be nil")
	assert.Equal(t, d.Name, rd.Name, "name should match")
	assert.Equal(t, d.Parent, rd.Parent, "parent id should match")

	oldUpdatedAt := rd.UpdatedAt

	rd.Name = "newtestdir"
	rd.Metadata = nil

	// update directory
	err = store.UpdateDirectory(context.Background(), rd)
	assert.NoError(t, err, "error querying db")
	assert.NotEqual(t, oldUpdatedAt, rd.UpdatedAt, "original directory should not have old UpdatedAt")

	// query database to ensure values updated
	var (
		newName      string
		newUpdatedAt time.Time
	)

	err = db.QueryRow("SELECT name, updated_at FROM directories WHERE id = $1", rd.Id).Scan(&newName, &newUpdatedAt)
	assert.NoError(t, err, "error querying db")

	assert.Equal(t, "newtestdir", newName, "database should have new name")
	assert.NotEqual(t, oldUpdatedAt, newUpdatedAt, "database should not have old UpdatedAt")
	assert.Equal(t, rd.UpdatedAt, newUpdatedAt, "database and directory should have matching UpdatedAt")

	// enable read only
	driver.WithReadOnly()(store)

	err = store.UpdateDirectory(context.Background(), rd)
	assert.Error(t, err, "read only error should've been returned")
}

func TestCreateDirectoryWithParentThatDoesntExist(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	pid := v1.DirectoryID(uuid.New())
	d := &v1.Directory{
		Name:   "testdir",
		Parent: &pid,
	}

	rd, err := store.CreateDirectory(context.Background(), d)
	assert.Error(t, err, "should have errored")
	assert.Nil(t, rd, "should be nil")
}

func TestCreateDirectoryWithoutParent(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	d := &v1.Directory{
		Name: "testdir",
	}

	rd, err := store.CreateDirectory(context.Background(), d)
	assert.Error(t, err, "should have errored")
	assert.Nil(t, rd, "should be nil")
}

func TestDeleteRootDirectory(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rootdir := withRootDir(t, store)

	child1 := &v1.Directory{
		Name:   "child1",
		Parent: &rootdir.Id,
	}

	_, err := store.CreateDirectory(context.Background(), child1)
	assert.NoError(t, err, "error creating child directory")

	affected, err := store.DeleteDirectory(context.Background(), rootdir.Id)
	assert.ErrorIs(t, err, storage.ErrDirectoryNotFound, "unexpected error returned")

	assert.Len(t, affected, 0, "should not have any affected records")
}

func TestDeleteDirectoryWithoutChildren(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rootdir := withRootDir(t, store)

	child1 := &v1.Directory{
		Name:   "child1",
		Parent: &rootdir.Id,
	}

	child1dir, err := store.CreateDirectory(context.Background(), child1)
	assert.NoError(t, err, "error creating child directory")

	affected, err := store.DeleteDirectory(context.Background(), child1dir.Id)
	assert.NoError(t, err, "error deleting directory")

	assert.Len(t, affected, 1, "should only have one affected row")
	assert.Equal(t, child1dir.Id, affected[0].Id, "affected id doesn't match child id")
	assert.NotNil(t, affected[0].DeletedAt, "DeletedAt should be set")

	// Ensure Getting deleted directory errors
	d, err := store.GetDirectory(context.Background(), child1dir.Id)
	assert.ErrorIs(t, err, storage.ErrDirectoryNotFound, "should have errored")
	assert.Nil(t, d, "deleted directory should not have been returned")

	// Ensure deleted directories are visible when using WithDeletedDirectories option
	d, err = store.GetDirectory(context.Background(), child1dir.Id, storage.WithDeletedDirectories)
	assert.NoError(t, err, "no error should have returned when using WithDeletedDirectories option")

	assert.NotNil(t, d, "directory returned should not be nil when using WithDeletedDirectories option")
	assert.Equal(t, child1.Id, d.Id, "id should match deleted directory")
}

func TestDeleteDirectoryWithChildren(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rootdir := withRootDir(t, store)

	child1 := &v1.Directory{
		Name:   "child1",
		Parent: &rootdir.Id,
	}

	child1dir, err := store.CreateDirectory(context.Background(), child1)
	assert.NoError(t, err, "error creating child 1 directory")

	child2 := &v1.Directory{
		Name:   "child2",
		Parent: &child1dir.Id,
	}

	child2dir, err := store.CreateDirectory(context.Background(), child2)
	assert.NoError(t, err, "error creating child 2 directory")

	affected, err := store.DeleteDirectory(context.Background(), child1dir.Id)
	assert.NoError(t, err, "error deleting directory")

	assert.Len(t, affected, 2, "should have two affected rows")

	for _, dir := range affected {
		switch dir.Id {
		case child1dir.Id:
			assert.NotNil(t, dir.DeletedAt, "child 1 DeletedAt should be set")
		case child2dir.Id:
			assert.NotNil(t, dir.DeletedAt, "child 2 DeletedAt should be set")
		default:
			t.Errorf("unexpected directory affected by deletion: %s", dir.Id)
		}
	}

	// Ensure getting deleted child directory errors
	children, err := store.GetChildren(context.Background(), child1dir.Id)
	assert.ErrorIs(t, err, storage.ErrDirectoryNotFound, "should have errored")
	assert.Len(t, children, 0, "no children should be returned for deleted directory")

	// Ensure deleted child directories are visible when using WithDeletedDirectories option
	children, err = store.GetChildren(context.Background(), child1dir.Id, storage.WithDeletedDirectories)
	assert.NoError(t, err, "no error should have returned when using WithDeletedDirectories option")

	assert.Len(t, children, 1, "children should have been returned when using WithDeletedDirectories option")
	assert.Equal(t, child2dir.Id, children[0], "id should match deleted directory")

	// Ensure getting deleted parent directory errors
	parents, err := store.GetParents(context.Background(), child2dir.Id)
	assert.ErrorIs(t, err, storage.ErrDirectoryNotFound, "should have errored")
	assert.Len(t, parents, 0, "no parents should be returned for deleted directory")

	// Ensure deleted parent directories are visible when using WithDeletedDirectories option
	parents, err = store.GetParents(context.Background(), child2dir.Id, storage.WithDeletedDirectories)
	assert.NoError(t, err, "no error should have returned when using WithDeletedDirectories option")

	assert.Len(t, parents, 2, "both parents should have been returned when using WithDeletedDirectories option")
	assert.Contains(t, parents, child1dir.Id, "parent id not found in list of parents")

	// Ensure deleted parent directories are visible when using WithDeletedDirectories option
	parents, err = store.GetParentsUntilAncestor(
		context.Background(),
		child2dir.Id, rootdir.Id,
		storage.WithDeletedDirectories,
	)
	assert.NoError(t, err, "no error should have returned when using WithDeletedDirectories option")

	assert.Len(t, parents, 2, "both parents should have been returned when using WithDeletedDirectories option")
	assert.Contains(t, parents, child1dir.Id, "parent id not found in list of parents")
}

func TestDeleteDirectoryAlreadyDeleted(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rootdir := withRootDir(t, store)

	child1 := &v1.Directory{
		Name:   "child1",
		Parent: &rootdir.Id,
	}

	child1dir, err := store.CreateDirectory(context.Background(), child1)
	assert.NoError(t, err, "error creating child directory")

	_, err = store.DeleteDirectory(context.Background(), child1dir.Id)
	assert.NoError(t, err, "error deleting directory")

	affected, err := store.DeleteDirectory(context.Background(), child1dir.Id)
	assert.ErrorIs(t, err, storage.ErrDirectoryNotFound, "unexpected error returned")

	assert.Len(t, affected, 0, "should not have any affected records")
}

func TestQueryUnknownDirectory(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	d, err := store.GetDirectory(context.Background(), v1.DirectoryID(uuid.New()))
	assert.ErrorIs(t, err, storage.ErrDirectoryNotFound, "should have errored")
	assert.Nil(t, d, "should be nil")
}

func TestGetSingleParent(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rootdir := withRootDir(t, store)

	d := &v1.Directory{
		Name:   "testdir",
		Parent: &rootdir.Id,
	}

	rd, err := store.CreateDirectory(context.Background(), d)
	assert.NoError(t, err, "error creating directory")

	parents, gperr := store.GetParents(context.Background(), rd.Id)
	assert.NoError(t, gperr, "error getting parents")
	assert.Len(t, parents, 1, "should have 1 parent")

	assert.Equal(t, rootdir.Id, parents[0], "id should match")
}

func TestGetMultipleParents(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rootdir := withRootDir(t, store)

	d1 := &v1.Directory{
		Name:   "testdir1",
		Parent: &rootdir.Id,
	}
	d2 := &v1.Directory{
		Name:   "testdir2",
		Parent: &d1.Id,
	}
	d3 := &v1.Directory{
		Name:   "testdir3",
		Parent: &d2.Id,
	}
	d4 := &v1.Directory{
		Name:   "testdir4",
		Parent: &d3.Id,
	}

	rd1, err := store.CreateDirectory(context.Background(), d1)
	assert.NoError(t, err, "error creating directory")

	_, err = store.CreateDirectory(context.Background(), d2)
	assert.NoError(t, err, "error creating directory")

	rd3, err := store.CreateDirectory(context.Background(), d3)
	assert.NoError(t, err, "error creating directory")

	rd4, err := store.CreateDirectory(context.Background(), d4)
	assert.NoError(t, err, "error creating directory")

	parents, gperr := store.GetParents(context.Background(), rd3.Id)
	assert.NoError(t, gperr, "error getting parents")
	assert.Len(t, parents, 3, "should have 3 parents")

	assert.Equal(t, d2.Id, parents[0], "id should match")
	assert.Equal(t, d1.Id, parents[1], "id should match")
	assert.Equal(t, rootdir.Id, parents[2], "id should match")

	parents, gperr = store.GetParentsUntilAncestor(context.Background(), rd4.Id, rd1.Id)
	assert.NoError(t, gperr, "error getting parents")
	assert.Len(t, parents, 3, "should have 3 parents")

	assert.Equal(t, d3.Id, parents[0], "id should match")
	assert.Equal(t, d2.Id, parents[1], "id should match")
	assert.Equal(t, d1.Id, parents[2], "id should match")
}

func TestGetParentFromRootDirShouldReturnEmpty(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rootdir := withRootDir(t, store)

	parents, gperr := store.GetParents(context.Background(), rootdir.Id)
	assert.NoError(t, gperr, "error getting parents")
	assert.Len(t, parents, 0, "should have 0 parents")
}

func TestGetParentsOfDeletedDirectory(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rootdir := withRootDir(t, store)

	d1 := &v1.Directory{
		Name:   "testdir1",
		Parent: &rootdir.Id,
	}
	d2 := &v1.Directory{
		Name:   "testdir2",
		Parent: &d1.Id,
	}

	_, err := store.CreateDirectory(context.Background(), d1)
	assert.NoError(t, err, "error creating directory")

	rd2, err := store.CreateDirectory(context.Background(), d2)
	assert.NoError(t, err, "error creating directory")

	_, err = store.DeleteDirectory(context.Background(), rd2.Id)
	assert.NoError(t, err, "deleting child directory")

	parents, gperr := store.GetParents(context.Background(), rd2.Id)
	assert.ErrorIs(t, gperr, storage.ErrDirectoryNotFound, "expect directory not found")
	assert.Len(t, parents, 0, "should have 0 parents")
}

func TestGetParentsFromUnknownShouldFail(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	parents, err := store.GetParents(context.Background(), v1.DirectoryID(uuid.New()))
	assert.ErrorIs(t, err, storage.ErrDirectoryNotFound, "should have errored")
	assert.Nil(t, parents, "should be nil")
}

func TestGetChildrenBasic(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rootdir := withRootDir(t, store)

	d := &v1.Directory{
		Name:   "testdir",
		Parent: &rootdir.Id,
	}

	rd, err := store.CreateDirectory(context.Background(), d)
	assert.NoError(t, err, "error creating directory")

	children, gperr := store.GetChildren(context.Background(), rootdir.Id)
	assert.NoError(t, gperr, "error getting children")
	assert.Len(t, children, 1, "should have 1 child")

	assert.Equal(t, rd.Id, children[0], "id should match")
}

func TestGetChildrenMultiple(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rootdir := withRootDir(t, store)

	d1 := &v1.Directory{
		Name:   "testdir1",
		Parent: &rootdir.Id,
	}

	d2 := &v1.Directory{
		Name:   "testdir2",
		Parent: &rootdir.Id,
	}

	d3 := &v1.Directory{
		Name:   "testdir3",
		Parent: &rootdir.Id,
	}

	d4 := &v1.Directory{
		Name:   "testdir4",
		Parent: &d1.Id,
	}

	rd1, err := store.CreateDirectory(context.Background(), d1)
	assert.NoError(t, err, "error creating directory")

	rd2, err := store.CreateDirectory(context.Background(), d2)
	assert.NoError(t, err, "error creating directory")

	rd3, err := store.CreateDirectory(context.Background(), d3)
	assert.NoError(t, err, "error creating directory")

	rd4, err := store.CreateDirectory(context.Background(), d4)
	assert.NoError(t, err, "error creating directory")

	children, gperr := store.GetChildren(context.Background(), rootdir.Id)
	assert.NoError(t, gperr, "error getting children")

	assert.Len(t, children, 4, "should have 4 children")

	assert.Contains(t, children, rd1.Id, "should contain id")
	assert.Contains(t, children, rd2.Id, "should contain id")
	assert.Contains(t, children, rd3.Id, "should contain id")
	assert.Contains(t, children, rd4.Id, "should contain id")
}

func TestGetChildrenMayReturnEmptyAppropriately(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rootdir := withRootDir(t, store)

	children, err := store.GetChildren(context.Background(), rootdir.Id)
	assert.NoError(t, err, "should have errored")
	assert.Len(t, children, 0, "should have 0 children")
}

func TestGetParentsUntilAncestorIsEmptyIfChildIsAncestor(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rootdir := withRootDir(t, store)

	ancestors, err := store.GetParentsUntilAncestor(context.Background(), rootdir.Id, rootdir.Id)
	assert.NoError(t, err, "should not have errored")
	assert.Len(t, ancestors, 0, "should have 0 children")
}

func TestGetParentsUntilAncestorParentNotFound(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	rootdir := withRootDir(t, store)

	ancestors, err := store.GetParentsUntilAncestor(context.Background(), v1.DirectoryID(uuid.New()), rootdir.Id)
	assert.ErrorIs(t, err, storage.ErrDirectoryNotFound, "should have errored")
	assert.Nil(t, ancestors, "should be nil")
}

func TestGetChildrenFromUnknownReturnsNotFound(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	store := driver.NewDirectoryDriver(db)

	children, err := store.GetChildren(context.Background(), v1.DirectoryID(uuid.New()))
	assert.ErrorIs(t, err, storage.ErrDirectoryNotFound, "should have errored")
	assert.Nil(t, children, "should be nil")
}

func TestOperationsFailWithBadDatabaseConnection(t *testing.T) {
	t.Parallel()

	// We're opening a valid database connection, but there's not database set.
	dbconn, err := sql.Open("postgres", baseDBURL.String())
	assert.NoError(t, err, "error creating db connection")

	store := driver.NewDirectoryDriver(dbconn)

	rd := &v1.Directory{
		Name: "testdir",
	}

	// Create root fails
	_, err = store.CreateRoot(context.Background(), rd)
	assert.Error(t, err, "should have errored")

	// get root fails
	_, err = store.ListRoots(context.Background())
	assert.Error(t, err, "should have errored")

	// Create directory fails
	someID := v1.DirectoryID(uuid.New())
	d := &v1.Directory{
		Name:   "testdir",
		Parent: &someID,
	}
	_, err = store.CreateDirectory(context.Background(), d)
	assert.Error(t, err, "should have errored")

	// Get directory fails
	_, err = store.GetDirectory(context.Background(), someID)
	assert.Error(t, err, "should have errored")

	// Get parents fails
	_, err = store.GetParents(context.Background(), someID)
	assert.Error(t, err, "should have errored")

	// Get children fails
	_, err = store.GetChildren(context.Background(), someID)
	assert.Error(t, err, "should have errored")

	// Get parents until ancestor fails
	otherID := v1.DirectoryID(uuid.New())
	_, err = store.GetParentsUntilAncestor(context.Background(), someID, otherID)
	assert.Error(t, err, "should have errored")
}
