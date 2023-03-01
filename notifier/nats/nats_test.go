package nats_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	natssrv "github.com/nats-io/nats-server/v2/server"
	natsgo "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/notifier/nats"
	natsutils "github.com/infratographer/fertilesoil/notifier/nats/utils"
)

const (
	natsMsgSubTimeout = 2 * time.Second
)

var natss *natssrv.Server

func TestMain(m *testing.M) {
	srv, err := natsutils.StartNatsServer()
	if err != nil {
		panic(err)
	}

	natss = srv

	defer natss.Shutdown()

	m.Run()
}

//nolint:paralleltest // Subtests must be sequential.
func TestBasicNotifications(t *testing.T) {
	subject := t.Name()

	conn, err := natsgo.Connect(natss.ClientURL())
	assert.NoError(t, err, "connecting to nats server")

	js, err := conn.JetStream()
	assert.NoError(t, err, "creating JetStream connection")

	clientconn, err := natsgo.Connect(natss.ClientURL())
	assert.NoError(t, err, "connecting to nats server")

	natsutils.WaitConnected(t, conn)
	natsutils.WaitConnected(t, clientconn)

	ntf := nats.NewNotifier(js, subject, nats.WithLogger(zaptest.NewLogger(t)))
	assert.NoError(t, err, "creating nats notifier")

	_, err = ntf.AddStream(&natsgo.StreamConfig{
		Name:    subject,
		Storage: natsgo.MemoryStorage,
	})
	assert.NoError(t, err, "creating JetStream stream")

	msgChan := make(chan *natsgo.Msg, 10)
	_, err = clientconn.Subscribe(subject+".*", func(m *natsgo.Msg) {
		t.Logf("Received message: %s", string(m.Data))
		msgChan <- m
	})
	assert.NoError(t, err, "creating NATS subscription")
	now := time.Now().UTC()
	dir := &apiv1.Directory{
		Id:   apiv1.DirectoryID(uuid.New()),
		Name: "test",
		Metadata: &apiv1.DirectoryMetadata{
			"foo": "bar",
		},
		CreatedAt: now,
		UpdatedAt: now,
		Parent:    nil,
	}

	t.Run("notify create", func(t *testing.T) {
		err = ntf.NotifyCreate(context.Background(), dir)
		assert.NoError(t, err, "notifying create")

		var msg *natsgo.Msg

		// Receive create
		select {
		case msg = <-msgChan:
		case <-time.After(natsMsgSubTimeout):
			t.Error("failed to receive nats message")
		}

		unmarshalled := &apiv1.DirectoryEvent{}
		err = json.Unmarshal(msg.Data, unmarshalled)

		assert.NoError(t, err, "unmarshalling nats message")
		assert.Equal(t, apiv1.EventTypeCreate, unmarshalled.Type)
		assert.Equal(t, dir.Id, unmarshalled.Directory.Id)
		assert.Equal(t, dir.Name, unmarshalled.Directory.Name)
		assert.Equal(t, dir.Metadata, unmarshalled.Directory.Metadata)
		assert.Equal(t, dir.CreatedAt, unmarshalled.Directory.CreatedAt)
	})

	t.Run("send update", func(t *testing.T) {
		now = time.Now().UTC()
		dir.UpdatedAt = time.Now().UTC()

		err = ntf.NotifyUpdate(context.Background(), dir)
		assert.NoError(t, err, "notifying update")

		var msg *natsgo.Msg

		// Receive update
		select {
		case msg = <-msgChan:
		case <-time.After(natsMsgSubTimeout):
			t.Error("failed to receive nats message")
		}

		unmarshalled := &apiv1.DirectoryEvent{}
		err = json.Unmarshal(msg.Data, unmarshalled)

		assert.NoError(t, err, "unmarshalling nats message")
		assert.Equal(t, apiv1.EventTypeUpdate, unmarshalled.Type)
		assert.Equal(t, dir.Id, unmarshalled.Directory.Id)
		assert.Equal(t, dir.Name, unmarshalled.Directory.Name)
		assert.Equal(t, dir.Metadata, unmarshalled.Directory.Metadata)
		assert.Equal(t, dir.UpdatedAt, unmarshalled.Directory.UpdatedAt)
	})

	t.Run("send delete", func(t *testing.T) {
		dir.DeletedAt = &now

		err = ntf.NotifyDelete(context.Background(), dir)
		assert.NoError(t, err, "notifying delete")

		var msg *natsgo.Msg

		// Receive delete
		select {
		case msg = <-msgChan:
		case <-time.After(natsMsgSubTimeout):
			t.Error("failed to receive nats message")
		}

		unmarshalled := &apiv1.DirectoryEvent{}
		err = json.Unmarshal(msg.Data, unmarshalled)

		assert.NoError(t, err, "unmarshalling nats message")
		assert.Equal(t, apiv1.EventTypeDelete, unmarshalled.Type)
		assert.Equal(t, dir.Id, unmarshalled.Directory.Id)
		assert.Equal(t, dir.Name, unmarshalled.Directory.Name)
		assert.Equal(t, dir.Metadata, unmarshalled.Directory.Metadata)
		assert.Equal(t, dir.DeletedAt, unmarshalled.Directory.DeletedAt)
	})

	t.Run("send hard delete", func(t *testing.T) {
		err = ntf.NotifyDeleteHard(context.Background(), dir)
		assert.NoError(t, err, "notifying delete hard")

		var msg *natsgo.Msg

		// Receive delete hard
		select {
		case msg = <-msgChan:
		case <-time.After(natsMsgSubTimeout):
			t.Error("failed to receive nats message")
		}

		unmarshalled := &apiv1.DirectoryEvent{}
		err = json.Unmarshal(msg.Data, unmarshalled)

		assert.NoError(t, err, "unmarshalling nats message")
		assert.Equal(t, apiv1.EventTypeDeleteHard, unmarshalled.Type)
	})
}

func TestNotifyCreateFailsOnBadConnection(t *testing.T) {
	t.Parallel()

	subject := t.Name()

	conn, err := natsgo.Connect(natss.ClientURL())
	assert.NoError(t, err, "connecting to nats server")

	js, err := conn.JetStream()
	assert.NoError(t, err, "creating JetStream connection")

	natsutils.WaitConnected(t, conn)

	ntf := nats.NewNotifier(js, subject, nats.WithLogger(zaptest.NewLogger(t)))

	_, err = ntf.AddStream(&natsgo.StreamConfig{
		Name:    subject,
		Storage: natsgo.MemoryStorage,
	})
	assert.NoError(t, err, "creating JetStream stream")

	// Send create
	dir := &apiv1.Directory{
		Id:   apiv1.DirectoryID(uuid.New()),
		Name: "test",
		Metadata: &apiv1.DirectoryMetadata{
			"foo": "bar",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Parent:    nil,
	}

	// Close the connection
	conn.Close()

	// Send create
	err = ntf.NotifyCreate(context.Background(), dir)
	assert.Error(t, err, "notifying create")
}

func TestAddStream(t *testing.T) {
	t.Parallel()

	subject := t.Name()

	conn, err := natsgo.Connect(natss.ClientURL())
	assert.NoError(t, err, "connecting to nats server")

	js, err := conn.JetStream()
	assert.NoError(t, err, "creating JetStream connection")

	natsutils.WaitConnected(t, conn)

	ntf := nats.NewNotifier(js, subject, nats.WithLogger(zaptest.NewLogger(t)))

	// Add new stream
	streamConfig := &natsgo.StreamConfig{
		Name:    "test",
		Storage: natsgo.MemoryStorage,
	}
	stream, err := ntf.AddStream(streamConfig)
	assert.NoError(t, err, "expected no error")
	assert.NotNil(t, stream, "expected stream to not be nil")
	assert.Equal(t, "test", stream.Config.Name, "created stream name doesn't match request")
	assert.Contains(t, stream.Config.Subjects, "TestAddStream.>", "expected subject to be added")

	// The provided Stream Config subjects are updated before adding the stream,
	// so we can check if the subject was added when it actually creates the stream.
	assert.Contains(t, streamConfig.Subjects, "TestAddStream.>", "expected subject to be added")

	// Use existing stream
	streamConfig = &natsgo.StreamConfig{
		Name:    "test",
		Storage: natsgo.MemoryStorage,
	}
	stream, err = ntf.AddStream(streamConfig)
	assert.NoError(t, err, "expected no error")
	assert.NotNil(t, stream, "expected stream to not be nil")
	assert.Equal(t, "test", stream.Config.Name, "created stream name doesn't match request")
	assert.Contains(t, stream.Config.Subjects, "TestAddStream.>", "expected subject to be added")

	// The provided Stream Config subjects are updated before adding the stream,
	// so we can check if the subject was added when it actually creates the stream.
	// In this case we expect it to discover an existing stream and not add the stream,
	// so the original streamConfig subjects should not contain the injected subject.
	assert.NotContains(t, streamConfig.Subjects, "TestAddStream.>", "expected subject to not be added")

	// Test StreamInfo failing for something other than Stream not found.
	streamConfig = &natsgo.StreamConfig{
		Name:    "bad.stream", // (.) and ( ) are not valid stream names
		Storage: natsgo.MemoryStorage,
	}
	stream, err = ntf.AddStream(streamConfig)
	assert.Error(t, err, "expected error to be returned for bad name")
	assert.Nil(t, stream, "expected stream to be nil")
}
