package treemanager

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	v1 "github.com/infratographer/fertilesoil/api/v1"
	"github.com/infratographer/fertilesoil/internal/httpsrv/common"
	"github.com/infratographer/fertilesoil/notifier"
	"github.com/infratographer/fertilesoil/notifier/noop"
	"github.com/infratographer/fertilesoil/storage"
	"github.com/infratographer/fertilesoil/storage/crdb/driver"
	sn "github.com/infratographer/fertilesoil/storage/notifier"
)

func NewServer(
	logger *zap.Logger,
	listen string,
	db *sql.DB,
	debug bool,
	shutdownTime time.Duration,
	unix string,
	n notifier.Notifier,
) *common.Server {
	dbdrv := driver.NewDirectoryDriver(db)

	var notif notifier.Notifier

	if n == nil {
		notif = noop.NewNotifier()
	} else {
		notif = n
	}

	store := sn.StorageWithNotifier(dbdrv, notif, sn.WithNotifyRetrier())
	s := common.NewServer(logger, listen, db, store, debug, shutdownTime, unix)

	s.SetHandler(newHandler(logger, s))

	return s
}

func newHandler(logger *zap.Logger, s *common.Server) *gin.Engine {
	r := s.DefaultEngine(logger)

	r.GET("/api", apiVersionHandler)
	r.GET("/api/v1", apiVersionHandler)

	r.GET("/api/v1/roots", listRoots(s))
	r.POST("/api/v1/roots", createRootDirectory(s))

	r.GET("/api/v1/directories/:id", getDirectory(s))
	r.POST("/api/v1/directories/:id", createDirectory(s))

	r.GET("/api/v1/directories/:id/children", listChildren(s))
	r.GET("/api/v1/directories/:id/parents", listParents(s))
	r.GET("/api/v1/directories/:id/parents/:until", listParentsUntil(s))

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
			DirectoryRequestMeta: v1.DirectoryRequestMeta{
				Version: v1.APIVersion,
			},
			Directories: roots,
		})
	}
}

func createRootDirectory(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.CreateDirectoryRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		d := v1.Directory{
			Name:     req.Name,
			Metadata: req.Metadata,
		}

		rd, err := s.T.CreateRoot(c, &d)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, &v1.DirectoryFetch{
			DirectoryRequestMeta: v1.DirectoryRequestMeta{
				Version: v1.APIVersion,
			},
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
			DirectoryRequestMeta: v1.DirectoryRequestMeta{
				Version: v1.APIVersion,
			},
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
		if err := c.BindJSON(&req); err != nil {
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

		parentID := parent.ID
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
			DirectoryRequestMeta: v1.DirectoryRequestMeta{
				Version: v1.APIVersion,
			},
			Directory: *rd,
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

		children, err := s.T.GetChildren(c, dir.ID)
		if err != nil {
			s.L.Error("error listing children", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusOK, &v1.DirectoryList{
			DirectoryRequestMeta: v1.DirectoryRequestMeta{
				Version: v1.APIVersion,
			},
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

		parents, err := s.T.GetParents(c, dir.ID)
		if err != nil {
			s.L.Error("error listing parents", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusOK, &v1.DirectoryList{
			DirectoryRequestMeta: v1.DirectoryRequestMeta{
				Version: v1.APIVersion,
			},
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

		parents, err := s.T.GetParentsUntilAncestor(c, dir.ID, untildir.ID)
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
