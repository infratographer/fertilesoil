package driver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	v1 "github.com/JAORMX/fertilesoil/api/v1"
	"github.com/JAORMX/fertilesoil/storage"
)

type Driver struct {
	db         *sql.DB
	readonly   bool
	rootaccess bool
}

func NewDirectoryAdminDriver(db *sql.DB) storage.DirectoryAdmin {
	return &Driver{
		db:         db,
		readonly:   false,
		rootaccess: true,
	}
}

func NewDirectoryReaderDriver(db *sql.DB) storage.Reader {
	return &Driver{
		db:         db,
		readonly:   true,
		rootaccess: false,
	}
}

// CreateRoot creates a root directory.
// Root directories are directories that have no parent directory.
// ID is generated by the database, it will be ignored if given.
func (t *Driver) CreateRoot(ctx context.Context, d *v1.Directory) (*v1.Directory, error) {
	if t.readonly {
		return nil, storage.ErrReadOnly
	}

	if !t.rootaccess {
		return nil, storage.ErrNoRootAccess
	}

	if d.Parent != nil {
		return nil, storage.ErrRootWithParentDirectory
	}

	if d.Metadata == nil {
		d.Metadata = v1.DirectoryMetadata{}
	}

	err := t.db.QueryRowContext(ctx,
		"INSERT INTO directories (name, metadata) VALUES ($1, $2) RETURNING id, created_at, updated_at",
		d.Name, d.Metadata).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("error inserting directory: %w", err)
	}

	return d, nil
}

func (t *Driver) ListRoots(ctx context.Context) ([]v1.DirectoryID, error) {
	if !t.rootaccess {
		return nil, storage.ErrNoRootAccess
	}

	var roots []v1.DirectoryID

	rows, err := t.db.QueryContext(ctx, "SELECT id FROM directories WHERE parent_id IS NULL")
	if err != nil {
		return nil, fmt.Errorf("error querying directory: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var did v1.DirectoryID
		err := rows.Scan(&did)
		if err != nil {
			return nil, fmt.Errorf("error scanning directory: %w", err)
		}
		roots = append(roots, did)
	}

	return roots, nil
}

func (t *Driver) CreateDirectory(ctx context.Context, d *v1.Directory) (*v1.Directory, error) {
	if t.readonly {
		return nil, storage.ErrReadOnly
	}

	if d.Parent == nil {
		return nil, storage.ErrDirectoryWithoutParent
	}

	if d.Metadata == nil {
		d.Metadata = v1.DirectoryMetadata{}
	}

	err := t.db.QueryRowContext(ctx,
		"INSERT INTO directories (name, parent_id, metadata) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at",
		d.Name, d.Parent.ID, d.Metadata).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("error inserting directory: %w", err)
	}

	return d, nil
}

// GetDirectoryByID returns a directory by its ID.
// Note that this call does not give out parent information.
func (t *Driver) GetDirectory(ctx context.Context, id v1.DirectoryID) (*v1.Directory, error) {
	var d v1.Directory
	err := t.db.QueryRowContext(ctx,
		"SELECT id, name, metadata, created_at, updated_at FROM directories WHERE id = $1",
		id).Scan(&d.ID, &d.Name, &d.Metadata, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrDirectoryNotFound
		}
		return nil, fmt.Errorf("error querying directory: %w", err)
	}

	return &d, nil
}

func (t *Driver) GetParents(ctx context.Context, child v1.DirectoryID) ([]v1.DirectoryID, error) {
	var parents []v1.DirectoryID

	// TODO(jaosorior): What's more efficient? A single recursive query or multiple queries?
	//                  Should we instead recurse in-code and do multiple queries?
	rows, err := t.db.QueryContext(ctx, `WITH RECURSIVE get_parents AS (
	SELECT id, parent_id FROM directories WHERE id = $1

	UNION

	SELECT d.id, d.parent_id FROM directories d
	INNER JOIN get_parents gp ON d.id = gp.parent_id
)
SELECT id FROM get_parents`, child)
	if err != nil {
		return nil, fmt.Errorf("error querying directory: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var did v1.DirectoryID
		err := rows.Scan(&did)
		if err != nil {
			return nil, fmt.Errorf("error scanning directory: %w", err)
		}
		parents = append(parents, did)
	}

	if len(parents) == 0 {
		return nil, storage.ErrDirectoryNotFound
	}

	// skip the first element, which is the child
	return parents[1:], nil
}

func (t *Driver) GetParentsUntilAncestor(ctx context.Context, child v1.DirectoryID, ancestor v1.DirectoryID) ([]v1.DirectoryID, error) {
	var parents []v1.DirectoryID

	// TODO(jaosorior): What's more efficient? A single recursive query or multiple queries?
	//                  Should we instead recurse in-code and do multiple queries?
	rows, err := t.db.QueryContext(ctx, `WITH RECURSIVE get_parents AS (
	SELECT id, parent_id FROM directories WHERE id = $1

	UNION

	SELECT d.id, d.parent_id FROM directories d
	INNER JOIN get_parents gp ON d.id = gp.parent_id
	WHERE gp.id != $2
) SELECT id FROM get_parents`, child, ancestor)
	if err != nil {
		return nil, fmt.Errorf("error querying directory: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var did v1.DirectoryID
		err := rows.Scan(&did)
		if err != nil {
			return nil, fmt.Errorf("error scanning directory: %w", err)
		}
		parents = append(parents, did)
	}

	if len(parents) == 0 {
		return nil, storage.ErrDirectoryNotFound
	}

	// skip the first element, which is the child
	return parents[1:], nil
}

func (t *Driver) GetChildren(ctx context.Context, parent v1.DirectoryID) ([]v1.DirectoryID, error) {
	var children []v1.DirectoryID

	rows, err := t.db.QueryContext(ctx, `WITH RECURSIVE get_children AS (
	SELECT id, parent_id FROM directories WHERE parent_id = $1

	UNION

	SELECT d.id, d.parent_id FROM directories d
	INNER JOIN get_children gc ON d.parent_id = gc.id
)
SELECT id FROM get_children`, parent)
	if err != nil {
		return nil, fmt.Errorf("error querying directory: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var did v1.DirectoryID
		err := rows.Scan(&did)
		if err != nil {
			return nil, fmt.Errorf("error scanning directory: %w", err)
		}
		children = append(children, did)
	}

	return children, nil
}
