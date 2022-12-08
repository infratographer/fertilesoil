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
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.infratographer.com/x/ginx"
	"go.infratographer.com/x/loggingx"
	"go.infratographer.com/x/viperx"
	"go.uber.org/zap"

	"github.com/JAORMX/fertilesoil/internal/httpsrv/treemanager"
	dbutils "github.com/JAORMX/fertilesoil/storage/crdb/utils"
)

const (
	defaultListen                = ":8080"
	defaultServerShutdownTimeout = 5 * time.Second
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
	ginx.MustViperFlags(v, serveCmd.Flags(), defaultListen)
	loggingx.MustViperFlags(v, serveCmd.Flags())

	// Server flags
	flags := serveCmd.Flags()
	flags.Duration("server-shutdown-timeout", defaultServerShutdownTimeout, "Time to wait for the server to shutdown gracefully")
	viperx.MustBindFlag(v, "server.shutdown", flags.Lookup("server-shutdown-timeout"))
	flags.String("server-unix-socket", "", "Listen on a unix socket instead of a TCP socket.")
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

	s := treemanager.NewServer(
		l,
		viper.GetString("listen"),
		db,
		viper.GetBool("debug"),
		viper.GetDuration("server.shutdown"),
		viper.GetString("server.unix_socket"),
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
