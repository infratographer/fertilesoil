// Code generated by ent, DO NOT EDIT.

package migrate

import (
	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/schema/field"
)

var (
	// DirectoriesColumns holds the columns for the "directories" table.
	DirectoriesColumns = []*schema.Column{
		{Name: "id", Type: field.TypeInt, Increment: true},
		{Name: "name", Type: field.TypeString},
		{Name: "metadata", Type: field.TypeString, Nullable: true},
		{Name: "created_at", Type: field.TypeTime},
		{Name: "updated_at", Type: field.TypeTime},
		{Name: "deleted_at", Type: field.TypeTime, Nullable: true},
		{Name: "is_root", Type: field.TypeBool, Default: false},
		{Name: "directory_children", Type: field.TypeInt, Nullable: true},
	}
	// DirectoriesTable holds the schema information for the "directories" table.
	DirectoriesTable = &schema.Table{
		Name:       "directories",
		Columns:    DirectoriesColumns,
		PrimaryKey: []*schema.Column{DirectoriesColumns[0]},
		ForeignKeys: []*schema.ForeignKey{
			{
				Symbol:     "directories_directories_children",
				Columns:    []*schema.Column{DirectoriesColumns[7]},
				RefColumns: []*schema.Column{DirectoriesColumns[0]},
				OnDelete:   schema.SetNull,
			},
		},
		Indexes: []*schema.Index{
			{
				Name:    "directory_id_is_root",
				Unique:  true,
				Columns: []*schema.Column{DirectoriesColumns[0], DirectoriesColumns[6]},
			},
		},
	}
	// Tables holds all the tables in the schema.
	Tables = []*schema.Table{
		DirectoriesTable,
	}
)

func init() {
	DirectoriesTable.ForeignKeys[0].RefTable = DirectoriesTable
}