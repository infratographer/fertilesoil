package treemanager

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	v1 "github.com/JAORMX/fertilesoil/api/v1"
	"github.com/JAORMX/fertilesoil/internal/httpsrv/common"
	"github.com/JAORMX/fertilesoil/storage"
	"github.com/JAORMX/fertilesoil/storage/crdb/driver"
)

func NewServer(
	logger *zap.Logger,
	listen string,
	db *sql.DB,
	debug bool,
	shutdownTime time.Duration,
	unix string,
) *common.Server {
	t := driver.NewDirectoryAdminDriver(db)
	s := common.NewServer(logger, listen, db, t, debug, shutdownTime, unix)

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

		d := v1.Directory{
			Name:     req.Name,
			Metadata: req.Metadata,
			Parent:   parent,
		}

		rd, err := s.T.CreateDirectory(c, &d)
		if errors.Is(err, storage.ErrDirectoryWithoutParent) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "directory must have a parent directory",
			})
			return
		} else if err != nil {
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

		c.JSON(http.StatusOK, gin.H{
			"directories": children,
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

		c.JSON(http.StatusOK, gin.H{
			"directories": parents,
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
