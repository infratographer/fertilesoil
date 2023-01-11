package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.infratographer.com/x/crdbx"

	"github.com/infratographer/fertilesoil/storage/crdb/migrations"
	dbutils "github.com/infratographer/fertilesoil/storage/crdb/utils"
)

// migrateCmd represents the migrate command.
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Executes a database migration",
	Long:  `Executes a database migration based on the current version of the database.`,
	RunE:  migrate,
}

//nolint:gochecknoinits // This is a Cobra generated file
func init() {
	rootCmd.AddCommand(migrateCmd)

	v := viper.GetViper()
	flags := migrateCmd.Flags()

	crdbx.MustViperFlags(v, flags)
}

func migrate(cmd *cobra.Command, args []string) error {
	v := viper.GetViper()

	// Initialize logger
	l := initLogger()

	// Initialize database connection
	dbconn, err := dbutils.GetDBConnection(v, "directory", false)
	if err != nil {
		return fmt.Errorf("failed to get db connection: %w", err)
	}

	l.Info("executing migrations")

	return migrations.Migrate(dbconn)
}
