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

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/notifier/nats"
	natsutils "github.com/infratographer/fertilesoil/notifier/nats/utils"
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

func TestBasicNotifications(t *testing.T) {
	t.Parallel()

	subject := t.Name()

	conn, err := natsgo.Connect(natss.ClientURL())
	assert.NoError(t, err, "connecting to nats server")

	js, err := conn.JetStream()
	assert.NoError(t, err, "creating JetStream connection")

	_, err = js.AddStream(&natsgo.StreamConfig{
		Name:     subject,
		Subjects: []string{subject},
		Storage:  natsgo.MemoryStorage,
	})
	assert.NoError(t, err, "creating JetStream stream")

	clientconn, err := natsgo.Connect(natss.ClientURL())
	assert.NoError(t, err, "connecting to nats server")

	natsutils.WaitConnected(t, conn)
	natsutils.WaitConnected(t, clientconn)

	ntf := nats.NewNotifier(js, subject)
	assert.NoError(t, err, "creating nats notifier")

	msgChan := make(chan *natsgo.Msg, 10)
	_, err = clientconn.Subscribe(subject, func(m *natsgo.Msg) {
		t.Logf("Received message: %s", string(m.Data))
		msgChan <- m
	})
	assert.NoError(t, err, "creating NATS subscription")

	// Send create
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

	err = ntf.NotifyCreate(context.Background(), dir)
	assert.NoError(t, err, "notifying create")

	// Receive create
	msg := <-msgChan

	unmarshalled := &apiv1.DirectoryEvent{}
	err = json.Unmarshal(msg.Data, unmarshalled)

	assert.NoError(t, err, "unmarshalling nats message")
	assert.Equal(t, apiv1.EventTypeCreate, unmarshalled.Type)
	assert.Equal(t, dir.Id, unmarshalled.Directory.Id)
	assert.Equal(t, dir.Name, unmarshalled.Directory.Name)
	assert.Equal(t, dir.Metadata, unmarshalled.Directory.Metadata)
	assert.Equal(t, dir.CreatedAt, unmarshalled.Directory.CreatedAt)

	// Send update
	now = time.Now().UTC()
	dir.UpdatedAt = time.Now().UTC()

	err = ntf.NotifyUpdate(context.Background(), dir)
	assert.NoError(t, err, "notifying update")

	// Receive update
	msg = <-msgChan

	unmarshalled = &apiv1.DirectoryEvent{}
	err = json.Unmarshal(msg.Data, unmarshalled)

	assert.NoError(t, err, "unmarshalling nats message")
	assert.Equal(t, apiv1.EventTypeUpdate, unmarshalled.Type)
	assert.Equal(t, dir.Id, unmarshalled.Directory.Id)
	assert.Equal(t, dir.Name, unmarshalled.Directory.Name)
	assert.Equal(t, dir.Metadata, unmarshalled.Directory.Metadata)
	assert.Equal(t, dir.UpdatedAt, unmarshalled.Directory.UpdatedAt)

	// Send delete
	dir.DeletedAt = &now

	err = ntf.NotifyDelete(context.Background(), dir)
	assert.NoError(t, err, "notifying delete")

	// Receive delete
	msg = <-msgChan

	unmarshalled = &apiv1.DirectoryEvent{}
	err = json.Unmarshal(msg.Data, unmarshalled)

	assert.NoError(t, err, "unmarshalling nats message")
	assert.Equal(t, apiv1.EventTypeDelete, unmarshalled.Type)
	assert.Equal(t, dir.Id, unmarshalled.Directory.Id)
	assert.Equal(t, dir.Name, unmarshalled.Directory.Name)
	assert.Equal(t, dir.Metadata, unmarshalled.Directory.Metadata)
	assert.Equal(t, dir.DeletedAt, unmarshalled.Directory.DeletedAt)

	// Send delete hard
	err = ntf.NotifyDeleteHard(context.Background(), dir)
	assert.NoError(t, err, "notifying delete hard")

	// Receive delete hard
	msg = <-msgChan

	unmarshalled = &apiv1.DirectoryEvent{}
	err = json.Unmarshal(msg.Data, unmarshalled)

	assert.NoError(t, err, "unmarshalling nats message")
	assert.Equal(t, apiv1.EventTypeDeleteHard, unmarshalled.Type)
}

func TestNotifyCreateFailsOnBadConnection(t *testing.T) {
	t.Parallel()

	subject := t.Name()

	conn, err := natsgo.Connect(natss.ClientURL())
	assert.NoError(t, err, "connecting to nats server")

	js, err := conn.JetStream()
	assert.NoError(t, err, "creating JetStream connection")

	_, err = js.AddStream(&natsgo.StreamConfig{
		Name:     subject,
		Subjects: []string{subject},
		Storage:  natsgo.MemoryStorage,
	})
	assert.NoError(t, err, "creating JetStream stream")

	natsutils.WaitConnected(t, conn)

	ntf := nats.NewNotifier(js, subject)

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
