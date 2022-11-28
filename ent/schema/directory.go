package schema

import (
	"context"
	"errors"
	"time"

	"entgo.io/contrib/entproto"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	gen "github.com/JAORMX/fertilesoil/ent"
	"github.com/JAORMX/fertilesoil/ent/directory"
	"github.com/JAORMX/fertilesoil/ent/hook"
)

// Directories are nodes in a tree structure.
type Directory struct {
	ent.Schema
}

// Fields of the Directory.
func (Directory) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty().Annotations(entproto.Field(2)),
		// NOTE(jaosorior): Can't use JSON if we're using protobufs
		field.String("metadata").Optional().Annotations(entproto.Field(3)),
		field.Time("created_at").Default(time.Now).Immutable().Annotations(entproto.Field(4)),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Annotations(entproto.Field(5)),
		field.Time("deleted_at").Optional().Nillable().Annotations(entproto.Field(6)),
		field.Bool("is_root").Default(false).Immutable().Annotations(entproto.Field(7)),
	}
}

func (Directory) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("children", Directory.Type).Annotations(entproto.Field(8)),
		edge.From("parent", Directory.Type).Ref("children").Unique().Annotations(entproto.Field(9)),
	}
}

func (Directory) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entproto.Message(),
		entproto.Service(
			entproto.Methods(entproto.MethodCreate | entproto.MethodGet | entproto.MethodList | entproto.MethodUpdate | entproto.MethodDelete),
		),
	}
}

func (Directory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("id", "is_root").Unique(),
	}
}

func (Directory) Hooks() []ent.Hook {
	return []ent.Hook{
		// Validate creation
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return hook.DirectoryFunc(func(ctx context.Context, dm *gen.DirectoryMutation) (ent.Value, error) {
					// Root Directory should be unique
					if isroot, exists := dm.IsRoot(); isroot && exists {
						// If this is the root, we need to make sure there's no other root
						// already in the database.
						rootcount, err := dm.Client().Directory.Query().Where(directory.IsRoot(true)).Count(ctx)
						// TODO(jaosorior): Should we retry?
						if err != nil {
							return nil, err
						}

						if rootcount > 0 {
							return nil, errors.New("there's already a root directory")
						}

						// root directory can't have a parent
						if _, exists := dm.ParentID(); exists {
							return nil, errors.New("root directory can't have a parent")
						}

						// A directory can't point to itself as a child
						if err := noRecursive(ctx, dm); err != nil {
							return nil, err
						}

						return next.Mutate(ctx, dm)
					}

					// Children directories should have a parent
					if _, exists := dm.ParentID(); !exists {
						return nil, errors.New("children directories should have a parent")
					}

					// A directory can't point to itself as a child
					if err := noRecursive(ctx, dm); err != nil {
						return nil, err
					}

					return next.Mutate(ctx, dm)
				})
			},
			ent.OpCreate,
		),
		// validate updates
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return hook.DirectoryFunc(func(ctx context.Context, dm *gen.DirectoryMutation) (ent.Value, error) {
					// Don't update parent of root directory
					if isroot, exists := dm.IsRoot(); isroot && exists {
						if _, exists := dm.ParentID(); exists {
							return nil, errors.New("root directory can't have a parent")

						}

						// A directory can't point to itself as a child
						if err := noRecursive(ctx, dm); err != nil {
							return nil, err
						}

						return next.Mutate(ctx, dm)
					}

					// children directories can't be orphaned
					if _, exists := dm.ParentID(); !exists {
						return nil, errors.New("children directories should have a parent")
					}

					// A directory can't point to itself as a child
					if err := noRecursive(ctx, dm); err != nil {
						return nil, err
					}

					return next.Mutate(ctx, dm)
				})
			},
			ent.OpUpdateOne,
		),
		// Validate deletion
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return hook.DirectoryFunc(func(ctx context.Context, dm *gen.DirectoryMutation) (ent.Value, error) {
					// Don't delete root directory
					if isroot, exists := dm.IsRoot(); isroot && exists {
						return nil, errors.New("root directory can't be deleted")
					}
					return next.Mutate(ctx, dm)
				})
			},
			ent.OpDeleteOne,
		),
		// We don't allow bulk updates nor deletes
		hook.Reject(ent.OpUpdate | ent.OpDelete),
	}
}

func noRecursive(ctx context.Context, dm *gen.DirectoryMutation) error {
	// A directory can't point to itself as a child
	if id, exists := dm.ID(); exists {
		if childrenids := dm.ChildrenIDs(); len(childrenids) > 0 {
			for _, childid := range childrenids {
				if childid == id {
					return errors.New("a directory can't point to itself as a child")
				}
			}
		}

		if parentid, exists := dm.ParentID(); exists {
			if parentid == id {
				return errors.New("a directory can't point to itself as a parent")
			}
		}
	}

	return nil
}
