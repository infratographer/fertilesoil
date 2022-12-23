/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.infratographer.com/x/ginx"
	"go.infratographer.com/x/loggingx"
	"go.infratographer.com/x/viperx"
	"go.uber.org/zap"

	"github.com/infratographer/fertilesoil/internal/httpsrv/treemanager"
	"github.com/infratographer/fertilesoil/notifier/nats"
	natsutils "github.com/infratographer/fertilesoil/notifier/nats/utils"
	"github.com/infratographer/fertilesoil/storage/crdb/driver"
	dbutils "github.com/infratographer/fertilesoil/storage/crdb/utils"
)

// serveCmd represents the treemanager command.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: serverRunE,
}

//nolint:gochecknoinits // This is encouraged by cobra
func init() {
	rootCmd.AddCommand(serveCmd)

	v := viper.GetViper()
	dbutils.RegisterDBArgs(v, serveCmd.Flags())
	ginx.MustViperFlags(v, serveCmd.Flags(), treemanager.DefaultTreeManagerListen)
	loggingx.MustViperFlags(v, serveCmd.Flags())
	natsutils.RegisterNATSArgs(v, serveCmd.Flags())

	// TODO(jaosorior): Add tracing
	// TODO(jaosorior): Add metrics
	// TODO(jaosorior): Add TLS flags

	// Server flags
	flags := serveCmd.Flags()

	// server shutdown timeout
	flags.Duration("server-shutdown-timeout",
		treemanager.DefaultTreeManagerShutdownTimeout,
		"Time to wait for the server to shutdown gracefully")
	viperx.MustBindFlag(v, "server.shutdown", flags.Lookup("server-shutdown-timeout"))

	// server UNIX socket
	flags.String("server-unix-socket",
		treemanager.DefaultTreeManagerUnix,
		"Listen on a unix socket instead of a TCP socket.")
	viperx.MustBindFlag(v, "server.unix_socket", flags.Lookup("server-unix-socket"))
}

func serverRunE(cmd *cobra.Command, args []string) error {
	l := initLogger()
	//nolint:errcheck // We don't care about the error here.
	// These logs aren't important enough to fail the program.
	defer l.Sync()

	// catch SIGTERM and SIGINT
	ctx, cancel := context.WithCancel(cmd.Context())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
		cancel()
	}()

	// TODO(jaosorior): Add tracing
	db, dberr := dbutils.GetDBConnection(viper.GetViper(), "directory", false)
	if dberr != nil {
		return dberr
	}

	v := viper.GetViper()

	natconn, natserr := natsutils.BuildNATSConnFromArgs(v)
	if natserr != nil {
		return natserr
	}

	subj := natsutils.BuildNATSSubject(v)

	notif, notiferr := nats.NewNotifier(natconn, subj)
	if notiferr != nil {
		return notiferr
	}

	store := driver.NewDirectoryDriver(db, dbutils.WithStorageOptions(v)...)

	s := treemanager.NewServer(
		l,
		db,
		treemanager.WithListen(v.GetString("server.listen")),
		treemanager.WithUnix(v.GetString("server.unix_socket")),
		treemanager.WithDebug(v.GetBool("debug")),
		treemanager.WithShutdownTimeout(v.GetDuration("server.shutdown")),
		treemanager.WithNotifier(notif),
		treemanager.WithStorageDriver(store),
	)

	go func() {
		if err := s.Run(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.Error("server error", zap.Error(err))
		}
	}()

	select {
	case <-c:
		cancel()
	case <-ctx.Done():
	}

	if err := s.Shutdown(); err != nil {
		l.Fatal("server forced to shutdown", zap.Error(err))
	}
	return nil
}

func initLogger() *zap.Logger {
	sl := loggingx.InitLogger("treemanager", loggingx.Config{
		Debug:  viper.GetBool("debug"),
		Pretty: viper.GetBool("pretty"),
	})

	return sl.Desugar()
}
