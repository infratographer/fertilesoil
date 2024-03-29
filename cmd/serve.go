/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/metal-toolbox/auditevent/ginaudit"
	"github.com/metal-toolbox/auditevent/helpers"
	natsgo "github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.hollow.sh/toolbox/ginjwt"
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
	ginjwt.RegisterViperOIDCFlags(v, serveCmd)
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

	// server trusted proxies
	flags.StringSlice("trusted-proxies", []string{}, "Proxy ips to trust X-Forwarded-* headers from")
	viperx.MustBindFlag(v, "server.trusted-proxies", flags.Lookup("trusted-proxies"))

	// audit log path
	flags.String("audit-log-path", "/app-audit/audit.log", "Path to the audit log file")
	viperx.MustBindFlag(v, "audit.log.path", flags.Lookup("audit-log-path"))
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

	auditLogPath := v.GetString("audit.log.path")
	fd, err := helpers.OpenAuditLogFileUntilSuccessWithContext(ctx, auditLogPath)
	if err != nil {
		return fmt.Errorf("failed to open audit log file: %w", err)
	}
	// The file descriptor shall be closed only if the gin server is shut down
	defer fd.Close()

	// Set up middleware with the file descriptor
	mdw := ginaudit.NewJSONMiddleware("tree-manager", fd)

	natconn, natserr := natsutils.BuildNATSConnFromArgs(v)
	if natserr != nil {
		return natserr
	}

	defer natconn.Close()

	natjs, err := natconn.JetStream()
	if err != nil {
		return err
	}

	subj := natsutils.BuildNATSSubject(v)

	notif := nats.NewNotifier(natjs, subj, nats.WithLogger(l))

	initNats(l, v, notif)

	authConfig := buildAuthConfig(v)

	store := driver.NewDirectoryDriver(db, dbutils.WithStorageOptions(v)...)

	s := treemanager.NewServer(
		l,
		db,
		treemanager.WithListen(v.GetString("server.listen")),
		treemanager.WithUnix(v.GetString("server.unix_socket")),
		treemanager.WithDebug(v.GetBool("debug")),
		treemanager.WithShutdownTimeout(v.GetDuration("server.shutdown")),
		treemanager.WithTrustedProxies(v.GetStringSlice("server.trusted-proxies")),
		treemanager.WithNotifier(notif),
		treemanager.WithStorageDriver(store),
		treemanager.WithAuditMiddleware(mdw),
		treemanager.WithAuthConfig(authConfig),
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

func buildAuthConfig(v *viper.Viper) *ginjwt.AuthConfig {
	var (
		jwksuri string
		issuer  string
	)

	jwksuris := v.GetStringSlice("oidc.jwksuri")

	if len(jwksuris) != 0 {
		jwksuri = jwksuris[0]
	}

	issuers := v.GetStringSlice("oidc.issuer")

	if len(issuers) != 0 {
		issuer = issuers[0]
	}

	authConfig := &ginjwt.AuthConfig{
		Enabled:                v.GetBool("oidc.enabled"),
		Audience:               v.GetString("oidc.audience"),
		Issuer:                 issuer,
		JWKSURI:                jwksuri,
		JWKSRemoteTimeout:      v.GetDuration("oidc.jwksremotetimeout"),
		RoleValidationStrategy: ginjwt.RoleValidationStrategy(v.GetString("oidc.rolevalidationstrategy")),
		RolesClaim:             v.GetString("oidc.claims.roles"),
		UsernameClaim:          v.GetString("oidc.claims.username"),
	}

	return authConfig
}

// initNats will call the NATS Notifier AddStream if stream_name is provided.
// If it already exists, nothing is done.
// If it's missing, it will be created with the provided config.
// The subject is automatically added by AddStream.
func initNats(logger *zap.Logger, v *viper.Viper, notifier *nats.Notifier) {
	if streamName := v.GetString("nats.stream_name"); streamName != "" {
		storage := natsgo.FileStorage
		if storageType := v.GetString("nats.stream_storage"); storageType == "memory" {
			storage = natsgo.MemoryStorage
		}

		_, err := notifier.AddStream(&natsgo.StreamConfig{
			Name:      streamName,
			Storage:   storage,
			Retention: natsgo.LimitsPolicy,
			Discard:   natsgo.DiscardNew,
		})
		if err != nil {
			logger.Fatal("failed to check or create stream", zap.Error(err))
		}
	}
}
