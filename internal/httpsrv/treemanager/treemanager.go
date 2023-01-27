package treemanager

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/metal-toolbox/auditevent/ginaudit"
	"go.hollow.sh/toolbox/ginjwt"
	"go.uber.org/zap"

	v1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/internal/httpsrv/common"
	"github.com/infratographer/fertilesoil/storage"
	sn "github.com/infratographer/fertilesoil/storage/notifier"
)

func NewServer(
	logger *zap.Logger,
	db *sql.DB,
	opts ...Option,
) *common.Server {
	cfg := &treeManagerConfig{
		listen:          DefaultTreeManagerListen,
		unix:            DefaultTreeManagerUnix,
		debug:           DefaultTreeManagerDebug,
		shutdownTimeout: DefaultTreeManagerShutdownTimeout,
		notif:           DefaultTreeManagerNotifier,
	}
	cfg.apply(opts...)

	store := sn.StorageWithNotifier(cfg.storageDriver, cfg.notif, sn.WithNotifyRetrier())

	s := common.NewServer(logger, cfg.listen, db, store, cfg.debug, cfg.shutdownTimeout, cfg.unix)

	s.SetHandler(newHandler(logger, s, cfg.auditMdw, cfg.authConfig))

	return s
}

func newHandler(
	logger *zap.Logger,
	s *common.Server,
	auditMdw *ginaudit.Middleware,
	authConfig *ginjwt.AuthConfig,
) *gin.Engine {
	r := s.DefaultEngine(logger)

	if auditMdw != nil {
		r.Use(auditMdw.Audit())
	}

	if authConfig == nil {
		authConfig = &ginjwt.AuthConfig{}
	}

	authMW, err := ginjwt.NewAuthMiddleware(*authConfig)
	if err != nil {
		logger.Fatal("failed to initialize auth middleware", zap.Error(err))
	}

	r.GET("/api", apiVersionHandler)
	r.GET("/api/v1", apiVersionHandler)

	r.GET("/api/v1/roots", authMW.AuthRequired(), listRoots(s))
	r.POST("/api/v1/roots", authMW.AuthRequired(), createRootDirectory(s))

	r.GET("/api/v1/directories/:id", authMW.AuthRequired(), getDirectory(s))
	r.POST("/api/v1/directories/:id", authMW.AuthRequired(), createDirectory(s))
	r.DELETE("/api/v1/directories/:id", authMW.AuthRequired(), deleteDirectory(s))

	r.GET("/api/v1/directories/:id/children", authMW.AuthRequired(), listChildren(s))
	r.GET("/api/v1/directories/:id/parents", authMW.AuthRequired(), listParents(s))
	r.GET("/api/v1/directories/:id/parents/:until", authMW.AuthRequired(), listParentsUntil(s))

	return r
}

func apiVersionHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		// NOTE(jaosorior): This is currently to v1.
		// As API versions come, we should change this.
		"version": "v1",
	})
}

func listRoots(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		roots, err := s.T.ListRoots(c)
		if err != nil {
			s.L.Error("error listing roots", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusOK, &v1.DirectoryList{
			Version:     v1.APIVersion,
			Directories: roots,
		})
	}
}

func createRootDirectory(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.CreateDirectoryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		d := v1.Directory{
			Name:     req.Name,
			Metadata: req.Metadata,
		}

		rd, err := s.T.CreateRoot(c, &d)
		if err != nil {
			s.L.Error("error creating root", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, &v1.DirectoryFetch{
			Version:   v1.APIVersion,
			Directory: *rd,
		})
	}
}

func getDirectory(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		idstr := c.Param("id")

		dir, err := getDirectoryFromReference(s.T, idstr)
		if err != nil {
			outputGetDirectoryError(c, err)
			return
		}

		c.JSON(http.StatusOK, &v1.DirectoryFetch{
			Version:   v1.APIVersion,
			Directory: *dir,
		})
	}
}

func createDirectory(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		idstr := c.Param("id")

		id, err := v1.ParseDirectoryID(idstr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid id",
			})
			return
		}

		var req v1.CreateDirectoryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var parent *v1.Directory
		parent, err = s.T.GetDirectory(c, id)
		if errors.Is(err, storage.ErrDirectoryNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "parent directory not found",
			})
			return
		} else if err != nil {
			s.L.Error("error getting directory", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		parentID := parent.Id
		d := v1.Directory{
			Name:     req.Name,
			Metadata: req.Metadata,
			Parent:   &parentID,
		}

		rd, err := s.T.CreateDirectory(c, &d)
		if err != nil {
			s.L.Error("error creating directory", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusCreated, &v1.DirectoryFetch{
			Version:   v1.APIVersion,
			Directory: *rd,
		})
	}
}

func deleteDirectory(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		idstr := c.Param("id")

		id, err := v1.ParseDirectoryID(idstr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid id",
			})
			return
		}

		affected, err := s.T.DeleteDirectory(c, id)
		if errors.Is(err, storage.ErrDirectoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "directory not found",
			})
			return
		} else if err != nil {
			s.L.Error("error deleting directory", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		affectedIDs := make([]v1.DirectoryID, len(affected))

		for i, d := range affected {
			affectedIDs[i] = d.Id
		}

		c.JSON(http.StatusOK, &v1.DirectoryList{
			Directories: affectedIDs,
			Version:     v1.APIVersion,
		})
	}
}

func listChildren(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		idstr := c.Param("id")

		dir, err := getDirectoryFromReference(s.T, idstr)
		if err != nil {
			outputGetDirectoryError(c, err)
			return
		}

		children, err := s.T.GetChildren(c, dir.Id)
		if err != nil {
			s.L.Error("error listing children", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusOK, &v1.DirectoryList{
			Version:     v1.APIVersion,
			Directories: children,
		})
	}
}

func listParents(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		idstr := c.Param("id")

		dir, err := getDirectoryFromReference(s.T, idstr)
		if err != nil {
			outputGetDirectoryError(c, err)
			return
		}

		parents, err := s.T.GetParents(c, dir.Id)
		if err != nil {
			s.L.Error("error listing parents", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusOK, &v1.DirectoryList{
			Version:     v1.APIVersion,
			Directories: parents,
		})
	}
}

func listParentsUntil(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		idstr := c.Param("id")

		dir, err := getDirectoryFromReference(s.T, idstr)
		if err != nil {
			outputGetDirectoryError(c, err)
			return
		}

		untilstr := c.Param("until")

		untildir, err := getDirectoryFromReference(s.T, untilstr)
		if err != nil {
			outputGetDirectoryError(c, err)
			return
		}

		parents, err := s.T.GetParentsUntilAncestor(c, dir.Id, untildir.Id)
		if err != nil {
			s.L.Error("error listing parents", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"directories": parents,
		})
	}
}

func getDirectoryFromReference(drv storage.DirectoryAdmin, idstr string) (*v1.Directory, error) {
	id, err := v1.ParseDirectoryID(idstr)
	if err != nil {
		return nil, err
	}

	dir, err := drv.GetDirectory(context.Background(), id)
	if err != nil {
		return nil, err
	}

	return dir, nil
}

func outputGetDirectoryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, v1.ErrParsingID):
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id",
		})

	case errors.Is(err, storage.ErrDirectoryNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error": "directory not found",
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
	}
}
