package driver_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	v1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/storage"
	"github.com/infratographer/fertilesoil/storage/crdb/driver"
	"github.com/infratographer/fertilesoil/storage/crdb/utils"
)

const (
	// ironically, testing fast reads is slow since we have to wait for
	// the follower read timestamp to be consistent with the latest data.
	waitConsistency = 5 * time.Second
)

var readonlyTestCases = []struct {
	name         string
	inconsistent bool // fast queries are inconsistent
}{
	{
		"consistent",
		false,
	},
	{
		"inconsistent",
		true,
	},
}

func TestReaderGetOneRoot(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	adminstore := driver.NewDirectoryDriver(db)

	rd := withRootDir(t, adminstore)

	for _, tc := range readonlyTestCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var store *driver.Driver

			opts := []driver.Options{driver.WithReadOnly()}
			if tc.inconsistent {
				opts = append(opts, driver.WithFastReads())
				time.Sleep(waitConsistency)
			}

			store = driver.NewDirectoryDriver(db, opts...)

			// retrieve from db
			queryrootdir, qerr := store.GetDirectory(context.Background(), rd.Id, nil)
			assert.NoError(t, qerr, "error querying db")
			assert.Equal(t, rd.Id, queryrootdir.Id, "id should match")
			assert.Equal(t, rd.Name, queryrootdir.Name, "name should match")

			// list roots returns only the root directory
			roots, err := store.ListRoots(context.Background(), nil)
			assert.NoError(t, err, "error listing roots")
			assert.Len(t, roots, 1, "should only be one root")
			assert.Equal(t, rd.Id, roots[0], "id should match")
		})
	}
}

func TestReaderGetOneDirectory(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	adminstore := driver.NewDirectoryDriver(db)

	rd := withRootDir(t, adminstore)

	// create a directory
	d := &v1.Directory{
		Name:   "testdir",
		Parent: &rd.Id,
	}

	retd, err := adminstore.CreateDirectory(context.Background(), d)
	assert.NoError(t, err, "error creating directory")
	assert.NotNil(t, retd.Metadata, "metadata should not be nil")
	assert.Equal(t, d.Name, retd.Name, "name should match")
	assert.Equal(t, d.Parent, retd.Parent, "parent id should match")

	for _, tc := range readonlyTestCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var store *driver.Driver

			opts := []driver.Options{driver.WithReadOnly()}
			if tc.inconsistent {
				opts = append(opts, driver.WithFastReads())
				time.Sleep(waitConsistency)
			}

			store = driver.NewDirectoryDriver(db, opts...)

			// retrieve from db
			querydir, qerr := store.GetDirectory(context.Background(), d.Id, nil)
			assert.NoError(t, qerr, "error querying db")
			assert.Equal(t, d.Id, querydir.Id, "id should match")

			// get as child of root
			children, err := store.GetChildren(context.Background(), rd.Id, nil)
			assert.NoError(t, err, "error getting children")
			assert.Len(t, children, 1, "should be one child")
			assert.Equal(t, d.Id, children[0], "id should match")

			// get parent
			parent, err := store.GetParents(context.Background(), d.Id, nil)
			assert.NoError(t, err, "error getting parents")
			assert.Len(t, parent, 1, "should be one parent")
			assert.Equal(t, rd.Id, parent[0], "id should match")
		})
	}
}

func TestReaderCannotCreateRootEvenIfCast(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	rostore := driver.NewDirectoryDriver(db, driver.WithReadOnly())

	d := &v1.Directory{
		Name: "root",
	}

	rd, err := rostore.CreateRoot(context.Background(), d)
	assert.ErrorIs(t, err, storage.ErrReadOnly, "error should be read only")
	assert.Nil(t, rd, "root directory should be nil")
}

func TestReaderCannotCreateDirectory(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	rostore := driver.NewDirectoryDriver(db, driver.WithReadOnly())

	d := &v1.Directory{
		Name: "testdir",
	}

	rd, err := rostore.CreateDirectory(context.Background(), d)
	assert.ErrorIs(t, err, storage.ErrReadOnly, "error should be read only")
	assert.Nil(t, rd, "directory should be nil")
}
