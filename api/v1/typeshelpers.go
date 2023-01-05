package v1

import (
	"errors"

	"github.com/google/uuid"
)

const APIVersion = "v1"

var ErrParsingID = errors.New("error parsing id")

func (d *Directory) IsRoot() bool {
	return d.Parent == nil
}

func (d *Directory) IsDeleted() bool {
	return d.DeletedAt != nil && !d.DeletedAt.IsZero()
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
