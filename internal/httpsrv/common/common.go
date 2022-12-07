package common

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/JAORMX/fertilesoil/storage/db/driver"
	"github.com/gin-gonic/gin"
	"go.infratographer.com/x/ginx"
	"go.infratographer.com/x/versionx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var defaultEmptyLogFn = func(c *gin.Context) []zapcore.Field { return []zapcore.Field{} }

type Server struct {
	DB              *sql.DB
	T               *driver.APIDBTransformer
	L               *zap.Logger
	debug           bool
	srv             *http.Server
	version         *versionx.Details
	readinessChecks map[string]ginx.CheckFunc
	shutdownTime    time.Duration
}

func NewServer(
	logger *zap.Logger,
	listen string,
	db *sql.DB,
	t *driver.APIDBTransformer,
	debug bool,
	shutdownTime time.Duration,
) *Server {
	srv := &http.Server{
		Addr: listen,
	}

	s := &Server{
		L:            logger,
		DB:           db,
		T:            t,
		debug:        debug,
		srv:          srv,
		shutdownTime: shutdownTime,
	}

	s.AddReadinessCheck("database", s.dbCheck)

	return s
}

func (s *Server) SetHandler(h http.Handler) {
	s.srv.Handler = h
}

// Run will start the server
func (s *Server) Run(ctx context.Context) error {
	if !s.debug {
		gin.SetMode(gin.ReleaseMode)
	}

	return s.srv.ListenAndServe()
}

// Shutdown will gracefully shutdown the server
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTime)
	defer cancel()

	return s.srv.Shutdown(ctx)
}

// AddReadinessCheck will accept a function to be ran during calls to /readyx.
// These functions should accept a context and only return an error. When adding
// a readiness check a name is also provided, this name will be used when returning
// the state of all the checks
func (s Server) AddReadinessCheck(name string, f ginx.CheckFunc) Server {
	s.readinessChecks[name] = f

	return s
}

// livenessCheckHandler ensures that the server is up and responding
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

func DefaultEngine(logger *zap.Logger) *gin.Engine {
	return ginx.DefaultEngine(logger, defaultEmptyLogFn)
}
