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
	tracked, err := s.IsDirectoryTracked(ctx, dir.Id)
	if err != nil {
		return false, err
	}

	if !tracked {
		return false, nil
	}

	var deletedAt sql.NullTime
	err = s.db.QueryRowContext(ctx,
		"SELECT deleted_at FROM tracked_directories WHERE id = $1",
		dir.Id).Scan(&deletedAt)
	if err != nil {
		return false, fmt.Errorf("error checking if directory exists: %w", err)
	}

	return compareDeletedAt(deletedAt, dir.DeletedAt), nil
}

func (s *sqlstorage) CreateDirectory(ctx context.Context, d *apiv1.Directory) (*apiv1.Directory, error) {
	// insert directory but ignore if conflict
	insertQuery := `INSERT INTO tracked_directories (id) VALUES ($1) ON CONFLICT DO NOTHING`
	res, err := s.db.ExecContext(ctx, insertQuery, d.Id)
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

func (s *sqlstorage) DeleteDirectory(ctx context.Context, id apiv1.DirectoryID) ([]*apiv1.Directory, error) {
	// soft delete directory
	var affected []*apiv1.Directory

	deleteQuery := `
		UPDATE tracked_directories
		SET deleted_at = NOW()
		WHERE
			deleted_at IS NULL
			AND id = $1
		RETURNING id, deleted_at`

	rows, err := s.db.QueryContext(ctx, deleteQuery, id)
	if err != nil {
		return nil, fmt.Errorf("error deleting directory: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var d apiv1.Directory

		err := rows.Scan(&d.Id, &d.DeletedAt)
		if err != nil {
			return nil, fmt.Errorf("error scanning directory: %w", err)
		}

		affected = append(affected, &d)
	}

	if len(affected) != 1 {
		return affected, fmt.Errorf("expected 1 row affected, got %d", len(affected))
	}

	return affected, nil
}

// compareDeletedAt compares the observed deleted at time with the expected deleted at time.
// It will return true if the observed and expected deleted at times are equal.
func compareDeletedAt(observed sql.NullTime, expected *time.Time) bool {
	if expected == nil {
		return !observed.Valid
	}

	if observed.Valid && expected.IsZero() {
		return false
	}

	if !observed.Valid && !expected.IsZero() {
		return false
	}

	if observed.Valid && !expected.IsZero() && observed.Time != *expected {
		return false
	}

	return true
}
