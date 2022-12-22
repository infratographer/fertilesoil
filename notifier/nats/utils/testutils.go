//go:build testtools
// +build testtools

package utils

import (
	"errors"
	"fmt"
	"testing"
	"time"

	natssrv "github.com/nats-io/nats-server/v2/server"
	natsgo "github.com/nats-io/nats.go"
)

const (
	tmout = 2 * time.Second
)

func StartNatsServer() (*natssrv.Server, error) {
	const maxControlLine = 2048

	s, err := natssrv.NewServer(&natssrv.Options{
		Host:           "127.0.0.1",
		Port:           natssrv.RANDOM_PORT,
		NoLog:          true,
		NoSigs:         true,
		MaxControlLine: maxControlLine,
	})
	if err != nil {
		return nil, fmt.Errorf("building nats server: %w", err)
	}

	//nolint:errcheck // we don't care about the error here
	go natssrv.Run(s)

	if !s.ReadyForConnections(tmout) {
		return nil, errors.New("starting nats server: timeout")
	}
	return s, nil
}

func WaitConnected(t *testing.T, c *natsgo.Conn) {
	t.Helper()

	const defaultWaitTime = 25 * time.Millisecond

	timeout := time.Now().Add(tmout)
	for time.Now().Before(timeout) {
		if c.IsConnected() {
			return
		}
		time.Sleep(defaultWaitTime)
	}
	t.Fatal("client connecting timeout")
}
