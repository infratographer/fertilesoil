package v1

import (
	"encoding/json"
	"io"
)

type DirectoryRequestMeta struct {
	Version string `json:"version"`
}

type CreateDirectoryRequest struct {
	DirectoryRequestMeta
	Name     string            `json:"name" binding:"required"`
	Metadata DirectoryMetadata `json:"metadata"`
}

type DirectoryList struct {
	DirectoryRequestMeta
	Directories []DirectoryID `json:"directories"`
}

func (ld *DirectoryList) Parse(r io.Reader) error {
	return json.NewDecoder(r).Decode(ld)
}

type DirectoryFetch struct {
	DirectoryRequestMeta
	Directory Directory `json:"directory"`
}

func (gd *DirectoryFetch) Parse(r io.Reader) error {
	return json.NewDecoder(r).Decode(gd)
}
