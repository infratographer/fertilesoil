package v1

import (
	"encoding/json"
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

func (did DirectoryID) MarshalJSON() ([]byte, error) {
	return json.Marshal(did.String())
}

func (did *DirectoryID) UnmarshalJSON(b []byte) error {
	var strUUID string
	if err := json.Unmarshal(b, &strUUID); err != nil {
		return err
	}

	u, err := uuid.Parse(strUUID)
	if err != nil {
		return err
	}

	*did = DirectoryID(u)

	return nil
}

type DirectoryMetadata map[string]string
