package utils

import (
	"database/sql"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.infratographer.com/x/crdbx"
	"go.infratographer.com/x/viperx"

	"github.com/infratographer/fertilesoil/storage/crdb/driver"
)

// RegisterDBArgs registers the arguments for the database connection.
func RegisterDBArgs(v *viper.Viper, flags *pflag.FlagSet) {
	crdbx.MustViperFlags(v, flags)

	// read-only mode
	flags.Bool("read-only", false, "Run the server in read-only mode.")
	viperx.MustBindFlag(v, "storage.read_only", flags.Lookup("read-only"))

	// fast reads
	flags.Bool("fast-reads", false, "Run the server in fast reads mode.")
	viperx.MustBindFlag(v, "storage.fast_reads", flags.Lookup("fast-reads"))
}

func GetDBConnection(v *viper.Viper, dbName string, tracing bool) (*sql.DB, error) {
	cfg := crdbx.ConfigFromArgs(v, dbName)
	return crdbx.NewDB(cfg, tracing)
}

// WithStorageOptions returns the storage options for the driver.
func WithStorageOptions(v *viper.Viper) []driver.Options {
	var opts []driver.Options
	if v.GetBool("storage.read_only") {
		opts = append(opts, driver.WithReadOnly())
	}

	if v.GetBool("storage.fast_reads") {
		opts = append(opts, driver.WithFastReads())
	}

	return opts
}
