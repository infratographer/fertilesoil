package v1

// EventType is the type of event that occurred.
type EventType string

const (
	EventTypeCreate     EventType = "create"
	EventTypeUpdate     EventType = "update"
	EventTypeDeleteSoft EventType = "deletesoft"
	EventTypeDeleteHard EventType = "deletehard"
)

// DirectoryEvent is the event that is sent to the event stream.
type DirectoryEvent struct {
	DirectoryRequestMeta
	Type      EventType `json:"type"`
	Time      string    `json:"time"`
	Directory Directory `json:"directory"`
}
