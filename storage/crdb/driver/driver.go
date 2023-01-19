package driver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	v1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/storage"
)

const (
	followerReadsQuery = "AS OF SYSTEM TIME follower_read_timestamp()"
)

type Driver struct {
	db        *sql.DB
	readOnly  bool
	fastReads bool
}

func NewDirectoryDriver(db *sql.DB, opts ...Options) *Driver {
	d := &Driver{
		db:        db,
		readOnly:  false,
		fastReads: false,
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

// CreateRoot creates a root directory.
// Root directories are directories that have no parent directory.
// ID is generated by the database, it will be ignored if given.
func (t *Driver) CreateRoot(ctx context.Context, d *v1.Directory) (*v1.Directory, error) {
	if t.readOnly {
		return nil, storage.ErrReadOnly
	}

	if d.Parent != nil {
		return nil, storage.ErrRootWithParentDirectory
	}

	if d.Metadata == nil {
		d.Metadata = &v1.DirectoryMetadata{}
	}

	err := t.db.QueryRowContext(ctx,
		"INSERT INTO directories (name, metadata) VALUES ($1, $2) RETURNING id, created_at, updated_at",
		d.Name, d.Metadata).Scan(&d.Id, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("error inserting directory: %w", err)
	}

	return d, nil
}

func (t *Driver) ListRoots(ctx context.Context) ([]v1.DirectoryID, error) {
	var roots []v1.DirectoryID

	q := t.formatQuery("SELECT id FROM directories %[1]s WHERE parent_id IS NULL AND deleted_at IS NULL")

	rows, err := t.db.QueryContext(ctx, q)
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
	if t.readOnly {
		return nil, storage.ErrReadOnly
	}

	if d.Parent == nil {
		return nil, storage.ErrDirectoryWithoutParent
	}

	if d.Metadata == nil {
		d.Metadata = &v1.DirectoryMetadata{}
	}

	err := t.db.QueryRowContext(ctx,
		"INSERT INTO directories (name, parent_id, metadata) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at",
		d.Name, d.Parent, d.Metadata).Scan(&d.Id, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("error inserting directory: %w", err)
	}

	return d, nil
}

func (t *Driver) DeleteDirectory(ctx context.Context, id v1.DirectoryID) ([]*v1.Directory, error) {
	var affected []*v1.Directory

	q := t.formatQuery(`WITH RECURSIVE get_children AS (
		SELECT id, parent_id FROM directories
		WHERE id = $1 AND deleted_at IS NULL AND parent_id IS NOT NULL

		UNION

		SELECT d.id, d.parent_id FROM directories d
		INNER JOIN get_children gc ON d.parent_id = gc.id
		WHERE d.deleted_at IS NULL
	)
	UPDATE directories
	SET deleted_at = NOW()
	WHERE
		deleted_at IS NULL
		AND id IN (SELECT id FROM get_children)
	RETURNING id, name, metadata, created_at, updated_at, deleted_at, parent_id %[1]s`)

	rows, err := t.db.QueryContext(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("error querying directory: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var d v1.Directory

		err := rows.Scan(&d.Id, &d.Name, &d.Metadata, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt, &d.Parent)
		if err != nil {
			return nil, fmt.Errorf("error scanning directory: %w", err)
		}

		affected = append(affected, &d)
	}

	// If no rows were affected, the directory wasn't found.
	if len(affected) == 0 {
		return nil, storage.ErrDirectoryNotFound
	}

	return affected, nil
}

// GetDirectoryByID returns a directory by its ID.
// Note that this call does not give out parent information.
func (t *Driver) GetDirectory(ctx context.Context, id v1.DirectoryID) (*v1.Directory, error) {
	var d v1.Directory

	q := t.formatQuery(`SELECT id, name, metadata, created_at, updated_at, deleted_at, parent_id FROM directories %[1]s
WHERE id = $1 AND deleted_at IS NULL`)

	err := t.db.QueryRowContext(ctx, q,
		id).Scan(&d.Id, &d.Name, &d.Metadata, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt, &d.Parent)
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
	q := t.formatQuery(`WITH RECURSIVE get_parents AS (
	SELECT id, parent_id FROM directories WHERE id = $1 AND deleted_at IS NULL

	UNION

	SELECT d.id, d.parent_id FROM directories d
	INNER JOIN get_parents gp ON d.id = gp.parent_id
	WHERE d.deleted_at IS NULL
)
SELECT id FROM get_parents %[1]s`)

	rows, err := t.db.QueryContext(ctx, q, child)
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

func (t *Driver) GetParentsUntilAncestor(
	ctx context.Context,
	child, ancestor v1.DirectoryID,
) ([]v1.DirectoryID, error) {
	// optimization: we don't need to go through the database
	// if the child is the ancestor
	if child == ancestor {
		return []v1.DirectoryID{}, nil
	}

	var parents []v1.DirectoryID

	// TODO(jaosorior): What's more efficient? A single recursive query or multiple queries?
	//                  Should we instead recurse in-code and do multiple queries?
	q := t.formatQuery(`WITH RECURSIVE get_parents AS (
	SELECT id, parent_id FROM directories
	WHERE id = $1 AND deleted_at IS NULL

	UNION

	SELECT d.id, d.parent_id FROM directories d
	INNER JOIN get_parents gp ON d.id = gp.parent_id
	WHERE gp.id != $2 AND d.deleted_at IS NULL
) SELECT id FROM get_parents %[1]s`)

	rows, err := t.db.QueryContext(ctx, q, child, ancestor)
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

	q := t.formatQuery(`WITH RECURSIVE get_children AS (
	SELECT id, parent_id FROM directories
	WHERE id = $1 AND deleted_at IS NULL

	UNION

	SELECT d.id, d.parent_id FROM directories d
	INNER JOIN get_children gc ON d.parent_id = gc.id
	WHERE d.deleted_at IS NULL
)
SELECT id FROM get_children %[1]s`)

	rows, err := t.db.QueryContext(ctx, q, parent)
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

	if len(children) == 0 {
		return nil, storage.ErrDirectoryNotFound
	}

	// skip the first element, which is the parent
	return children[1:], nil
}

// Note that this assumes that queries only take one
// formatting argument.
func (t *Driver) formatQuery(query string) string {
	if t.fastReads {
		return withFollowerReads(query)
	}

	return fmt.Sprintf(query, "")
}

func withFollowerReads(query string) string {
	return fmt.Sprintf(query, followerReadsQuery)
}
