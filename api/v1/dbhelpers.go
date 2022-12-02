package v1

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

func (did DirectoryID) Value() (driver.Value, error) {
	return uuid.UUID(did).String(), nil
}

func (did *DirectoryID) Scan(value interface{}) error {
	var u uuid.UUID
	err := u.Scan(value)
	if err != nil {
		return err
	}

	*did = DirectoryID(u)
	return nil
}

func (dm DirectoryMetadata) Value() (driver.Value, error) {
	return json.Marshal(dm)
}

func (dm *DirectoryMetadata) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &dm)
}
