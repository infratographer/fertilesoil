package utils

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/assert"
	"go.hollow.sh/toolbox/ginjwt"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	clientv1 "github.com/infratographer/fertilesoil/client/v1"
	"github.com/infratographer/fertilesoil/internal/httpsrv/common"
)

// NewUnixsocketPath returns a new unix socket path for testing.
func NewUnixsocketPath(t *testing.T) string {
	t.Helper()
	tmpdir := t.TempDir()
	skt := filepath.Join(tmpdir, "skt")
	return skt
}

func RunTestServer(t *testing.T, srv *common.Server) {
	t.Helper()

	err := srv.Run(context.Background())
	assert.ErrorIs(t, err, http.ErrServerClosed, "unexpected error running server")
}

func NewTestClient(t *testing.T, skt string, srvURL *url.URL, authConfig *ginjwt.AuthConfig) clientv1.HTTPRootClient {
	t.Helper()

	var client *http.Client

	if skt != "" {
		client = clientv1.UnixClient(skt)
	}

	client = getAuthClient(client, authConfig)

	cfg := clientv1.NewClientConfig().WithClient(client).WithManagerURL(srvURL)
	httpc := clientv1.NewHTTPRootClient(cfg)
	return httpc
}

func getAuthClient(client *http.Client, authConfig *ginjwt.AuthConfig) *http.Client {
	if authConfig != nil {
		ctx := context.Background()

		if client != nil {
			ctx = context.WithValue(ctx, oauth2.HTTPClient, client)
		}

		authClaim := jwt.Claims{
			Subject:   "test-user",
			Issuer:    authConfig.Issuer,
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Audience:  jwt.Audience{authConfig.Audience, "another.test.service"},
		}

		signer := ginjwt.TestHelperMustMakeSigner(jose.RS256, ginjwt.TestPrivRSAKey1ID, ginjwt.TestPrivRSAKey1)
		rawToken := ginjwt.TestHelperGetToken(signer, authClaim, "scope", "test")

		return oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: rawToken,
		}))
	}

	return client
}

func WaitForServer(t *testing.T, cli clientv1.HTTPClient) {
	t.Helper()

	const (
		maxRetries      = 10
		backoffDuration = 5 * time.Millisecond
	)

	err := backoff.Retry(func() error {
		readyz, err := cli.DoRaw(context.Background(), http.MethodGet, "/readyz", nil)
		if err != nil {
			return err
		}
		defer readyz.Body.Close()
		if readyz.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", readyz.StatusCode)
		}
		return nil
	}, backoff.WithMaxRetries(backoff.NewConstantBackOff(backoffDuration), maxRetries))
	assert.NoError(t, err, "error waiting for server to be ready")
}
