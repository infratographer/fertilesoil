//go:build testtools
// +build testtools

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	clientv1 "github.com/infratographer/fertilesoil/client/v1"
	"github.com/infratographer/fertilesoil/storage"
)

//nolint:thelper // In this case, we don't want to use t.Helper() because we want to see theline number of the caller.
func MalformedDataTest(t *testing.T, skt string, baseServerAddress *url.URL) {
	c := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", skt)
			},
		},
	}

	u := baseServerAddress.JoinPath("/api/v1/roots")

	// Invalid request with valid JSON
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	err := enc.Encode(map[string]string{"foo": "bar"})
	assert.NoError(t, err, "error encoding data")

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, u.String(), &buf)
	assert.NoError(t, err, "error creating request")

	resp, err := c.Do(req)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// invalid request with invalid JSON
	buf.Reset()
	buf.WriteString("{\"foo")

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, u.String(), &buf)
	assert.NoError(t, err, "error creating request")

	resp, err = c.Do(req)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()
}

//nolint:thelper // In this case, we don't want to use t.Helper() because we want to see the line number of the caller.
func InvalidDirectoryIDsTest(t *testing.T, cli clientv1.HTTPRootClient) {
	// Create a root directory
	root, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "root",
	})
	assert.NoError(t, err, "error creating root directory")

	// Create a child directory
	child, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "child",
	}, root.Directory.Id)
	assert.NoError(t, err, "error creating child directory")

	// some string
	resp, err := cli.DoRaw(context.Background(), http.MethodGet, "/api/v1/directories/invalid", nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// some string getting children
	resp, err = cli.DoRaw(context.Background(), http.MethodGet, "/api/v1/directories/invalid/children", nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// some string getting parents
	resp, err = cli.DoRaw(context.Background(), http.MethodGet, "/api/v1/directories/invalid/parents", nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// some string getting parents until
	resp, err = cli.DoRaw(context.Background(), http.MethodGet,
		"/api/v1/directories/invalid/parents/00000000-0000-0000-0000-000000000000", nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// a valid child with an invalid parent
	resp, err = cli.DoRaw(context.Background(), http.MethodGet,
		fmt.Sprintf("/api/v1/directories/%s/parents/invalid", child.Directory.Id),
		nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// almost valid UUID
	resp, err = cli.DoRaw(context.Background(), http.MethodGet,
		"/api/v1/directories/00000000-0000-0000-0000-00000000XXX/children", nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// crazy long string
	const stringLength = 1000
	resp, err = cli.DoRaw(context.Background(), http.MethodGet,
		fmt.Sprintf("/api/v1/directories/%s/children", strings.Repeat("a", stringLength)), nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// SQL injection through DirectoryID
	resp, err = cli.DoRaw(context.Background(), http.MethodGet,
		"/api/v1/directories/00000000-0000-0000-0000-000000000000; DROP TABLE directories", nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// SQL injection through DirectoryID (using valid root ID)
	resp, err = cli.DoRaw(context.Background(), http.MethodGet,
		fmt.Sprintf("/api/v1/directories/%s; DROP TABLE directories", root.Directory.Id),
		nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()
}

//nolint:thelper // In this case, we don't want to use t.Helper() because we want to see the line number of the caller.
func ErroneousDirectoryTest(t *testing.T, cli clientv1.HTTPRootClient) {
	rd, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "root",
	})
	assert.NoError(t, err, "error creating root")

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	err = enc.Encode(&apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "child",
	})
	assert.NoError(t, err, "error encoding data")

	// Creating directory with invalid parent
	resp, err := cli.DoRaw(context.Background(), http.MethodPost,
		"/api/v1/directories/invalid", &buf)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// creating directory with null request
	resp, err = cli.DoRaw(context.Background(), http.MethodPost,
		fmt.Sprintf("/api/v1/directories/%s", rd.Directory.Id), nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// creating directory with invalid request
	resp, err = cli.DoRaw(context.Background(), http.MethodPost,
		fmt.Sprintf("/api/v1/directories/%s", rd.Directory.Id), bytes.NewBufferString("invalid"))
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// creating directory with invalid request (no name)
	buf.Reset()
	err = enc.Encode(&apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
	})
	assert.NoError(t, err, "error encoding data")
	resp, err = cli.DoRaw(context.Background(), http.MethodPost,
		fmt.Sprintf("/api/v1/directories/%s", rd.Directory.Id), &buf)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// creating directory with parent that doesn't exist
	nodir, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "nodir",
	}, apiv1.DirectoryID(uuid.New()))
	assert.Error(t, err, "should have errored creating directory")
	assert.Nil(t, nodir, "directory should be nil")
}

//nolint:thelper // In this case, we don't want to use t.Helper() because we want to see the line number of the caller.
func DirectoryNotFoundTest(t *testing.T, cli clientv1.HTTPRootClient) {
	// unknown directory
	resp, err := cli.GetDirectory(context.Background(), apiv1.DirectoryID(uuid.New()))
	assert.Error(t, err, "should have errored getting directory")
	assert.Nil(t, resp, "directory should be nil")
}

//nolint:thelper // In this case, we don't want to use t.Helper() because we want to see the line number of the caller.
func DeleteDirectoryTest(t *testing.T, cli clientv1.HTTPRootClient) {
	rd, err := cli.CreateRoot(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "root",
	})
	assert.NoError(t, err, "error creating root")

	ch1, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "child1",
	}, rd.Directory.Id)
	assert.NoError(t, err, "error creating child1")

	ch2, err := cli.CreateDirectory(context.Background(), &apiv1.CreateDirectoryRequest{
		Version: apiv1.APIVersion,
		Name:    "child2",
	}, ch1.Directory.Id)
	assert.NoError(t, err, "error creating child2")

	// Delete directory with invalid parent
	resp, err := cli.DoRaw(context.Background(), http.MethodDelete,
		"/api/v1/directories/invalid", nil)
	assert.NoError(t, err, "error sending request")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	resp.Body.Close()

	// Delete root directory
	rootDelResp, err := cli.DeleteDirectory(context.Background(), rd.Directory.Id)
	assert.Error(t, err, "should have errored")
	assert.Nil(t, rootDelResp, "response should not be nil")

	// Delete child directory
	delResp, err := cli.DeleteDirectory(context.Background(), ch1.Directory.Id)
	assert.NoError(t, err, "should not have errored")
	assert.NotNil(t, delResp, "response should not be nil")
	assert.Len(t, delResp.Directories, 2, "expected 2 affected directories") //nolint:gomnd // runs as part of tests

	for _, id := range delResp.Directories {
		switch id {
		case ch1.Directory.Id:
		case ch2.Directory.Id:
		default:
			t.Errorf("unexpected directory affected by deletion: %s", id)
		}
	}

	// Check that deleted directory is not fetchable
	getResp, err := cli.GetDirectory(context.Background(), ch2.Directory.Id)
	assert.Error(t, err, "should have errored getting directory")
	assert.Nil(t, getResp, "directory should be nil")

	// Delete missing directory
	delDelResp, err := cli.DeleteDirectory(context.Background(), ch1.Directory.Id)
	assert.Error(t, err, "should have errored")
	assert.Nil(t, delDelResp, "expected nil affected directories")

	// Check that deleted directory is fetchable when WithDeleted is true
	getResp, err = cli.GetDirectory(context.Background(), ch2.Directory.Id, storage.WithDeletedDirectories)
	assert.NoError(t, err, "should not have errored getting directory")
	assert.Equal(t, ch2.Directory.Id, getResp.Directory.Id, "unexpected return directory id")

	// Check that deleted directory children are fetchable when WithDeleted is true
	listResp, err := cli.GetChildren(context.Background(), ch1.Directory.Id, storage.WithDeletedDirectories)
	assert.NoError(t, err, "should not have errored getting children")
	assert.Equal(t, 1, len(listResp.Directories), "unexpected number of children returned")
	assert.Equal(t, ch2.Directory.Id, listResp.Directories[0], "unexpected child directory id")
}
