package sql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
	appv1 "github.com/infratographer/fertilesoil/app/v1"
)

type sqlstorage struct {
	db *sql.DB
}

// implement AppStorage.
var _ appv1.AppStorage = (*sqlstorage)(nil)

func New(conn *sql.DB) appv1.AppStorage {
	return &sqlstorage{
		db: conn,
	}
}

func (s *sqlstorage) IsDirectoryTracked(ctx context.Context, id apiv1.DirectoryID) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM tracked_directories WHERE id = $1)",
		id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error checking if directory exists: %w", err)
	}

	return exists, nil
}

func (s *sqlstorage) IsDirectoryInfoUpdated(ctx context.Context, dir *apiv1.Directory) (bool, error) {
	// Verify if the directory is tracked and if the ID and deleted at info are up to date.
	tracked, err := s.IsDirectoryTracked(ctx, dir.ID)
	if err != nil {
		return false, err
	}

	if !tracked {
		return false, nil
	}

	var deletedAt sql.NullTime
	err = s.db.QueryRowContext(ctx,
		"SELECT deleted_at FROM tracked_directories WHERE id = $1",
		dir.ID).Scan(&deletedAt)
	if err != nil {
		return false, fmt.Errorf("error checking if directory exists: %w", err)
	}

	return compareDeletedAt(deletedAt, dir.DeletedAt), nil
}

func (s *sqlstorage) CreateDirectory(ctx context.Context, d *apiv1.Directory) (*apiv1.Directory, error) {
	// insert directory but ignore if conflict
	insertQuery := `INSERT INTO tracked_directories (id) VALUES ($1) ON CONFLICT DO NOTHING`
	res, err := s.db.ExecContext(ctx, insertQuery, d.ID)
	if err != nil {
		return nil, fmt.Errorf("error inserting directory: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("error getting rows affected: %w", err)
	}

	if rows != 1 {
		return nil, fmt.Errorf("expected 1 row affected, got %d", rows)
	}

	return d, nil
}

func (s *sqlstorage) DeleteDirectory(ctx context.Context, id apiv1.DirectoryID) error {
	// soft delete directory
	res, err := s.db.ExecContext(ctx, "UPDATE tracked_directories SET deleted_at = NOW() WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("error deleting directory: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rows != 1 {
		return fmt.Errorf("expected 1 row affected, got %d", rows)
	}

	return nil
}

func compareDeletedAt(observed sql.NullTime, expected time.Time) bool {
	if observed.Valid && expected.IsZero() {
		return false
	}

	if !observed.Valid && !expected.IsZero() {
		return false
	}

	if observed.Valid && !expected.IsZero() && observed.Time != expected {
		return false
	}

	return true
}
