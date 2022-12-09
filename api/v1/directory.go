package v1

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const APIVersion = "v1"

var ErrParsingID = errors.New("error parsing id")

type Directory struct {
	ID        DirectoryID       `json:"id"`
	Name      string            `json:"name"`
	Metadata  DirectoryMetadata `json:"metadata"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
	DeletedAt time.Time         `json:"deletedAt"`

	// Parent is the parent directory.
	// The visibility of this structure depends on the query.
	// Full tree queries are normally not allowed.
	Parent *DirectoryID `json:"parent,omitempty"`
}

type DirectoryID uuid.UUID

func ParseDirectoryID(id string) (DirectoryID, error) {
	u, err := uuid.Parse(id)
	if err != nil {
		return DirectoryID{}, ErrParsingID
	}

	return DirectoryID(u), nil
}

func (did DirectoryID) String() string {
	return uuid.UUID(did).String()
}

type DirectoryMetadata map[string]string
