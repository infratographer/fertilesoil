package common

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"go.infratographer.com/x/ginx"
	"go.infratographer.com/x/versionx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/infratographer/fertilesoil/storage"
)

var defaultEmptyLogFn = func(c *gin.Context) []zapcore.Field { return []zapcore.Field{} }

const (
	DefaultServerReadHeaderTimeout = 5 * time.Second
)

type Server struct {
	DB              *sql.DB
	T               storage.DirectoryAdmin
	L               *zap.Logger
	debug           bool
	srv             *http.Server
	listen          string
	listenUnix      string
	shutdownTime    time.Duration
	version         *versionx.Details
	readinessChecks map[string]ginx.CheckFunc
}

func NewServer(
	logger *zap.Logger,
	listen string,
	db *sql.DB,
	t storage.DirectoryAdmin,
	debug bool,
	shutdownTime time.Duration,
	unix string,
) *Server {
	srv := &http.Server{
		Addr:              listen,
		ReadHeaderTimeout: DefaultServerReadHeaderTimeout,
	}

	s := &Server{
		L:               logger,
		DB:              db,
		T:               t,
		debug:           debug,
		srv:             srv,
		listen:          listen,
		listenUnix:      unix,
		shutdownTime:    shutdownTime,
		readinessChecks: make(map[string]ginx.CheckFunc),
	}

	s.AddReadinessCheck("database", s.dbCheck)

	return s
}

func (s *Server) DefaultEngine(logger *zap.Logger) *gin.Engine {
	r := ginx.DefaultEngine(logger, defaultEmptyLogFn)
	p := ginprometheus.NewPrometheus("gin")

	// Remove any params from the URL string to keep the number of labels down
	p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
		return c.FullPath()
	}

	p.Use(r)

	if s.version != nil {
		// Version endpoint returns build information
		r.GET("/version", s.versionHandler)
	}

	// Health endpoints
	r.GET("/livez", s.livenessCheckHandler)
	r.GET("/readyz", s.readinessCheckHandler)

	r.Use(func(c *gin.Context) {
		u := c.GetHeader("User")
		if u != "" {
			c.Set("current_actor", u)
			c.Set("actor_type", "user")
		}
	})

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "invalid request - route not found"})
	})

	return r
}

func (s *Server) SetHandler(h http.Handler) {
	s.srv.Handler = h
}

// Run will start the server.
func (s *Server) Run(ctx context.Context) error {
	if !s.debug {
		gin.SetMode(gin.ReleaseMode)
	}

	if s.listenUnix != "" {
		s.L.Info("listening on unix socket", zap.String("socket", s.listenUnix))
		listener, err := net.Listen("unix", s.listenUnix)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", s.listenUnix, err)
		}
		defer listener.Close()
		defer os.Remove(s.listenUnix)

		return s.srv.Serve(listener)
	}

	s.L.Info("listening on", zap.String("address", s.listen))
	return s.srv.ListenAndServe()
}

// Shutdown will gracefully shutdown the server.
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTime)
	defer cancel()

	return s.srv.Shutdown(ctx)
}

// AddReadinessCheck will accept a function to be ran during calls to /readyx.
// These functions should accept a context and only return an error. When adding
// a readiness check a name is also provided, this name will be used when returning
// the state of all the checks.
func (s *Server) AddReadinessCheck(name string, f ginx.CheckFunc) *Server {
	s.readinessChecks[name] = f

	return s
}

// livenessCheckHandler ensures that the server is up and responding.
func (s *Server) livenessCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
	})
}

// readinessCheckHandler ensures that the server is up and that we are able to process
// requests. It will check any readinessChecks that have been provided and return
// their status when calculating if the service is ready.
func (s *Server) readinessCheckHandler(c *gin.Context) {
	failed := false
	status := map[string]string{}

	for name, check := range s.readinessChecks {
		if err := check(c.Request.Context()); err != nil {
			s.L.Sugar().Error("readiness check failed", "name", name, "error", err)

			failed = true
			status[name] = err.Error()
		} else {
			status[name] = "OK"
		}
	}

	if failed {
		c.JSON(http.StatusServiceUnavailable, status)
		return
	}

	c.JSON(http.StatusOK, status)
}

// version returns the version build information.
func (s *Server) versionHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.version)
}

func (s *Server) dbCheck(ctx context.Context) error {
	return s.DB.PingContext(ctx)
}
