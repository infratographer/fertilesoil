// Package v1 provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.12.4 DO NOT EDIT.
package v1

import (
	"time"
)

// CreateDirectoryRequest defines model for CreateDirectoryRequest.
type CreateDirectoryRequest struct {
	Metadata *DirectoryMetadata `json:"metadata,omitempty"`
	Name     string             `binding:"required" json:"name"`
	Version  string             `json:"version"`
}

// Directory defines model for Directory.
type Directory struct {
	CreatedAt time.Time          `json:"createdAt"`
	DeletedAt *time.Time         `json:"deletedAt,omitempty"`
	Id        DirectoryID        `json:"id"`
	Metadata  *DirectoryMetadata `json:"metadata,omitempty"`
	Name      string             `binding:"required" json:"name"`
	Parent    *DirectoryID       `json:"parent,omitempty"`
	UpdatedAt time.Time          `json:"updatedAt"`
}

// DirectoryFetch defines model for DirectoryFetch.
type DirectoryFetch struct {
	Directory Directory `json:"directory"`
	Version   string    `json:"version"`
}

// DirectoryList defines model for DirectoryList.
type DirectoryList struct {
	Directories []DirectoryID `json:"directories"`
	Version     string        `json:"version"`
}

// DirectoryRequestMeta defines model for DirectoryRequestMeta.
type DirectoryRequestMeta struct {
	Version string `json:"version"`
}

// Error defines model for Error.
type Error struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

// NewDirectory defines model for NewDirectory.
type NewDirectory struct {
	Metadata *DirectoryMetadata `json:"metadata,omitempty"`
	Name     string             `binding:"required" json:"name"`
}

// CreateDirectoryJSONRequestBody defines body for CreateDirectory for application/json ContentType.
type CreateDirectoryJSONRequestBody = CreateDirectoryRequest

// CreateRootDirectoryJSONRequestBody defines body for CreateRootDirectory for application/json ContentType.
type CreateRootDirectoryJSONRequestBody = CreateDirectoryRequest
