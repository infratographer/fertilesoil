package storage

import (
	"errors"
)

var (
	// ErrRootWithParentDirectory is returned when a root directory has a parent directory.
	ErrRootWithParentDirectory = errors.New("root directory cannot have parent directory")

	// ErrNoRowsAffected is returned when no rows are affected by an operation.
	ErrNoRowsAffected = errors.New("no rows affected")

	// ErrDirectoryWithoutParent is returned when a directory does not have a parent directory.
	ErrDirectoryWithoutParent = errors.New("directory must have a parent directory")

	// ErrDirectoryNotFound is returned when a directory is not found.
	ErrDirectoryNotFound = errors.New("directory not found")

	// ErrReadOnly is returned when a write operation is attempted on a read-only driver.
	ErrReadOnly = errors.New("attempted write operation on read-only driver")

	// ErrNoRootAccess is returned when a root directory is attempted to be accessed
	// without root access.
	ErrNoRootAccess = errors.New("attempted to access root directory without root access")
)
