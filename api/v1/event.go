package v1

import "time"

// EventType is the type of event that occurred.
type EventType string

const (
	EventTypeCreate     EventType = "create"
	EventTypeUpdate     EventType = "update"
	EventTypeDelete     EventType = "delete"
	EventTypeDeleteHard EventType = "deletehard"
)

// DirectoryEvent is the event that is sent to the event stream.
type DirectoryEvent struct {
	DirectoryRequestMeta
	Type      EventType `json:"type"`
	Time      time.Time `json:"time"`
	Directory Directory `json:"directory"`
}
