package driver_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/storage"
	"github.com/infratographer/fertilesoil/storage/crdb/driver"
	"github.com/infratographer/fertilesoil/storage/crdb/utils"
)

func TestReaderGetOneRoot(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	adminstore := driver.NewDirectoryAdminDriver(db)
	rostore := driver.NewDirectoryReaderDriver(db)

	rd := withRootDir(t, adminstore)

	// retrieve from db
	queryrootdir, qerr := rostore.GetDirectory(context.Background(), rd.ID)
	assert.NoError(t, qerr, "error querying db")
	assert.Equal(t, rd.ID, queryrootdir.ID, "id should match")
	assert.Equal(t, rd.Name, queryrootdir.Name, "name should match")
}

func TestReaderGetOneDirectory(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	adminstore := driver.NewDirectoryAdminDriver(db)
	rostore := driver.NewDirectoryReaderDriver(db)

	rd := withRootDir(t, adminstore)

	// create a directory
	d := &v1.Directory{
		Name:   "testdir",
		Parent: &rd.ID,
	}

	retd, err := adminstore.CreateDirectory(context.Background(), d)
	assert.NoError(t, err, "error creating directory")
	assert.NotNil(t, retd.Metadata, "metadata should not be nil")
	assert.Equal(t, d.Name, retd.Name, "name should match")
	assert.Equal(t, d.Parent, retd.Parent, "parent id should match")

	// retrieve from db
	querydir, qerr := rostore.GetDirectory(context.Background(), d.ID)
	assert.NoError(t, qerr, "error querying db")
	assert.Equal(t, d.ID, querydir.ID, "id should match")

	// get as child of root
	children, err := rostore.GetChildren(context.Background(), rd.ID)
	assert.NoError(t, err, "error getting children")
	assert.Len(t, children, 1, "should be one child")
	assert.Equal(t, d.ID, children[0], "id should match")

	// get parent
	parent, err := rostore.GetParents(context.Background(), d.ID)
	assert.NoError(t, err, "error getting parents")
	assert.Len(t, parent, 1, "should be one parent")
	assert.Equal(t, rd.ID, parent[0], "id should match")
}

func TestReaderCannotCreateRootEvenIfCast(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	rostore := driver.NewDirectoryReaderDriver(db)

	d := &v1.Directory{
		Name: "root",
	}

	admcast, ok := rostore.(storage.DirectoryAdmin)
	assert.True(t, ok, "reader should be castable to admin")

	rd, err := admcast.CreateRoot(context.Background(), d)
	assert.ErrorIs(t, err, storage.ErrReadOnly, "error should be read only")
	assert.Nil(t, rd, "root directory should be nil")
}

func TestReaderCannotCreateDirectoryEvenIfCast(t *testing.T) {
	t.Parallel()

	db := utils.GetNewTestDB(t, baseDBURL)
	rostore := driver.NewDirectoryReaderDriver(db)

	d := &v1.Directory{
		Name: "testdir",
	}

	admcast, ok := rostore.(storage.DirectoryAdmin)
	assert.True(t, ok, "reader should be castable to admin")

	rd, err := admcast.CreateDirectory(context.Background(), d)
	assert.ErrorIs(t, err, storage.ErrReadOnly, "error should be read only")
	assert.Nil(t, rd, "directory should be nil")
}
