package treemanager

import (
	"database/sql"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	v1 "github.com/JAORMX/fertilesoil/api/v1"
	"github.com/JAORMX/fertilesoil/internal/db/transformers"
	"github.com/JAORMX/fertilesoil/internal/httpsrv/common"
)

func NewServer(
	logger *zap.Logger,
	listen string,
	db *sql.DB,
	debug bool,
	shutdownTime time.Duration,
) *common.Server {
	t := transformers.NewAPIDBTransformer(db)
	s := common.NewServer(logger, listen, db, t, debug, shutdownTime)

	s.SetHandler(newHandler(logger, s))

	return s
}

func newHandler(logger *zap.Logger, s *common.Server) *gin.Engine {
	r := common.DefaultEngine(logger)

	r.GET("/api", apiVersionHandler)
	r.GET("/api/v1", apiVersionHandler)

	r.GET("/api/v1/roots", listRoots(s))
	r.POST("/api/v1/roots", createRootDirectory(s))

	r.GET("/api/v1/directories/:id", getDirectory(s))
	r.POST("/api/v1/directories/:id", createDirectory(s))

	r.GET("/api/v1/directories/:id/children", listChildren(s))
	r.GET("/api/v1/directories/:id/parents", listParents(s))

	return r
}

func apiVersionHandler(c *gin.Context) {
	c.JSON(200, gin.H{
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
			c.JSON(500, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(200, gin.H{
			"directories": roots,
		})
	}
}

func createRootDirectory(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.CreateDirectoryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		d := v1.Directory{
			Name:     req.Name,
			Metadata: req.Metadata,
		}

		rd, err := s.T.CreateRoot(c, &d)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, gin.H{
			"id": rd.ID,
		})
	}
}

func getDirectory(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		idstr := c.Param("id")

		id, err := v1.ParseDirectoryID(idstr)
		if err != nil {
			c.JSON(400, gin.H{
				"error": "invalid id",
			})
			return
		}

		dir, err := s.T.GetDirectory(c, id)
		if errors.Is(err, transformers.ErrDirectoryNotFound) {
			c.JSON(404, gin.H{
				"error": "directory not found",
			})
			return
		} else if err != nil {
			s.L.Error("error getting directory", zap.Error(err))
			c.JSON(500, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(200, gin.H{
			"directories": dir,
		})
	}
}

func createDirectory(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		idstr := c.Param("id")

		id, err := v1.ParseDirectoryID(idstr)
		if err != nil {
			c.JSON(400, gin.H{
				"error": "invalid id",
			})
			return
		}

		var req v1.CreateDirectoryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		var parent *v1.Directory
		parent, err = s.T.GetDirectory(c, id)
		if errors.Is(err, transformers.ErrDirectoryNotFound) {
			c.JSON(400, gin.H{
				"error": "parent directory not found",
			})
			return
		} else if err != nil {
			s.L.Error("error getting directory", zap.Error(err))
			c.JSON(500, gin.H{
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
		if errors.Is(err, transformers.ErrDirectoryWithoutParent) {
			c.JSON(400, gin.H{
				"error": "directory must have a parent directory",
			})
			return
		} else if err != nil {
			s.L.Error("error creating directory", zap.Error(err))
			c.JSON(500, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(201, gin.H{
			"id": rd.ID,
		})
	}
}

func listChildren(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		idstr := c.Param("id")

		id, err := v1.ParseDirectoryID(idstr)
		if err != nil {
			c.JSON(400, gin.H{
				"error": "invalid id",
			})
			return
		}

		dir, err := s.T.GetDirectory(c, id)
		if errors.Is(err, transformers.ErrDirectoryNotFound) {
			c.JSON(404, gin.H{
				"error": "directory not found",
			})
			return
		} else if err != nil {
			s.L.Error("error getting directory", zap.Error(err))
			c.JSON(500, gin.H{
				"error": "internal server error",
			})
			return
		}

		children, err := s.T.GetChildren(c, dir.ID)
		if err != nil {
			s.L.Error("error listing children", zap.Error(err))
			c.JSON(500, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(200, gin.H{
			"directories": children,
		})
	}
}

func listParents(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		idstr := c.Param("id")

		id, err := v1.ParseDirectoryID(idstr)
		if err != nil {
			c.JSON(400, gin.H{
				"error": "invalid id",
			})
			return
		}

		dir, err := s.T.GetDirectory(c, id)
		if errors.Is(err, transformers.ErrDirectoryNotFound) {
			c.JSON(404, gin.H{
				"error": "directory not found",
			})
			return
		} else if err != nil {
			s.L.Error("error getting directory", zap.Error(err))
			c.JSON(500, gin.H{
				"error": "internal server error",
			})
			return
		}

		parents, err := s.T.GetParents(c, dir.ID)
		if err != nil {
			s.L.Error("error listing parents", zap.Error(err))
			c.JSON(500, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(200, gin.H{
			"directories": parents,
		})
	}
}
