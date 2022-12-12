package driver_test

import (
	"context"
	"net/url"
	"testing"

	v1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/storage"
	"github.com/infratographer/fertilesoil/storage/crdb/utils"
	"github.com/stretchr/testify/assert"
)

var baseDBURL *url.URL

func TestMain(m *testing.M) {
	var stop func()
	baseDBURL, stop = utils.NewTestDBServerOrDie()
	defer stop()

	m.Run()
}

func withRootDir(t *testing.T, store storage.DirectoryAdmin) *v1.Directory {
	t.Helper()

	d := &v1.Directory{
		Name: "root",
	}

	rd, err := store.CreateRoot(context.Background(), d)
	assert.NoError(t, err, "error creating root directory")
	assert.NotNil(t, rd.Metadata, "metadata should not be nil")
	assert.Equal(t, d.Name, rd.Name, "name should match")

	return d
}
