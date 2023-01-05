package v1

import (
	"encoding/json"
	"io"
)

func (ld *DirectoryList) Parse(r io.Reader) error {
	return json.NewDecoder(r).Decode(ld)
}

func (gd *DirectoryFetch) Parse(r io.Reader) error {
	return json.NewDecoder(r).Decode(gd)
}
