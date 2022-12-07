package driver

import (
	"errors"
)

var (
	// ErrRootWithParentDirectory is returned when a root directory has a parent directory
	ErrRootWithParentDirectory = errors.New("root directory cannot have parent directory")

	// ErrNoRowsAffected is returned when no rows are affected by an operation
	ErrNoRowsAffected = errors.New("no rows affected")

	// ErrDirectoryWithoutParent is returned when a directory does not have a parent directory
	ErrDirectoryWithoutParent = errors.New("directory must have a parent directory")

	// ErrDirectoryNotFound is returned when a directory is not found
	ErrDirectoryNotFound = errors.New("directory not found")
)
