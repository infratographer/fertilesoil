package treemanager

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strconv"

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

	s := common.NewServer(
		logger,
		cfg.listen,
		db,
		store,
		cfg.debug,
		cfg.shutdownTimeout,
		cfg.unix,
		cfg.listener,
		cfg.trustedProxies,
	)

	s.SetHandler(newHandler(logger, s, cfg.auditMdw, cfg.authConfig))

	return s
}

func newHandler(
	logger *zap.Logger,
	s *common.Server,
	auditMdw *ginaudit.Middleware,
	authConfig *ginjwt.AuthConfig,
) *gin.Engine {
	r, err := s.DefaultEngine(logger)
	if err != nil {
		logger.Fatal("failed to initialize route engine", zap.Error(err))
	}

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
	r.PATCH("/api/v1/directories/:id", authMW.AuthRequired(), updateDirectory(s))
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
		options, err := storageOptionsFromGetQuery(c)
		if err != nil {
			s.L.Error("error building storage.ListOptions from GetQuery", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "bad request",
			})
			return
		}

		roots, err := s.T.ListRoots(c, options...)
		if err != nil {
			s.L.Error("error listing roots", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		pagination := paginationResponse(c, len(roots), options)

		c.JSON(http.StatusOK, &v1.DirectoryList{
			Version:     v1.APIVersion,
			Page:        pagination.Page,
			PageSize:    pagination.PageSize,
			Links:       pagination.Links,
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
		options, err := storageOptionsFromGetQuery(c)
		if err != nil {
			s.L.Error("error building storage.GetOptions from GetQuery", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "bad request",
			})
			return
		}

		idstr := c.Param("id")

		dir, err := getDirectoryFromReference(s.T, idstr, options...)
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

func updateDirectory(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		idstr := c.Param("id")

		id, err := v1.ParseDirectoryID(idstr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid id",
			})
			return
		}

		var req v1.UpdateDirectoryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		d, err := s.T.GetDirectory(c, id)
		if errors.Is(err, storage.ErrDirectoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "directory not found",
			})
			return
		}

		if req.Name != nil && *req.Name != "" {
			d.Name = *req.Name
		}

		if req.Metadata != nil {
			d.Metadata = req.Metadata
		}

		if err := s.T.UpdateDirectory(c, d); err != nil {
			s.L.Error("error updating directory", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to update directory",
			})
			return
		}

		c.JSON(http.StatusOK, &v1.DirectoryFetch{
			Directory: *d,
			Version:   v1.APIVersion,
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

//nolint:dupl // listChildren and listParents are very similar, but not the same.
func listChildren(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		options, err := storageOptionsFromGetQuery(c)
		if err != nil {
			s.L.Error("error building storage.ListOptions from GetQuery", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "bad request",
			})
			return
		}

		idstr := c.Param("id")

		dir, err := getDirectoryFromReference(s.T, idstr, options...)
		if err != nil {
			outputGetDirectoryError(c, err)
			return
		}

		children, err := s.T.GetChildren(c, dir.Id, options...)
		if err != nil {
			s.L.Error("error listing children", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		pagination := paginationResponse(c, len(children), options)

		c.JSON(http.StatusOK, &v1.DirectoryList{
			Version:     v1.APIVersion,
			Page:        pagination.Page,
			PageSize:    pagination.PageSize,
			Links:       pagination.Links,
			Directories: children,
		})
	}
}

//nolint:dupl // listChildren and listParents are very similar, but not the same.
func listParents(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		options, err := storageOptionsFromGetQuery(c)
		if err != nil {
			s.L.Error("error building storage.ListOptions from GetQuery", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "bad request",
			})
			return
		}

		idstr := c.Param("id")

		dir, err := getDirectoryFromReference(s.T, idstr, options...)
		if err != nil {
			outputGetDirectoryError(c, err)
			return
		}

		parents, err := s.T.GetParents(c, dir.Id, options...)
		if err != nil {
			s.L.Error("error listing parents", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		pagination := paginationResponse(c, len(parents), options)

		c.JSON(http.StatusOK, &v1.DirectoryList{
			Version:     v1.APIVersion,
			Page:        pagination.Page,
			PageSize:    pagination.PageSize,
			Links:       pagination.Links,
			Directories: parents,
		})
	}
}

func listParentsUntil(s *common.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		options, err := storageOptionsFromGetQuery(c)
		if err != nil {
			s.L.Error("error building storage.ListOptions from GetQuery", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "bad request",
			})
			return
		}

		idstr := c.Param("id")

		dir, err := getDirectoryFromReference(s.T, idstr, options...)
		if err != nil {
			outputGetDirectoryError(c, err)
			return
		}

		untilstr := c.Param("until")

		untildir, err := getDirectoryFromReference(s.T, untilstr, options...)
		if err != nil {
			outputGetDirectoryError(c, err)
			return
		}

		parents, err := s.T.GetParentsUntilAncestor(c, dir.Id, untildir.Id, options...)
		if err != nil {
			s.L.Error("error listing parents", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		pagination := paginationResponse(c, len(parents), options)

		c.JSON(http.StatusOK, &v1.DirectoryList{
			Version:     v1.APIVersion,
			Page:        pagination.Page,
			PageSize:    pagination.PageSize,
			Links:       pagination.Links,
			Directories: parents,
		})
	}
}

func getDirectoryFromReference(
	drv storage.DirectoryAdmin,
	idstr string,
	options ...storage.Option,
) (*v1.Directory, error) {
	id, err := v1.ParseDirectoryID(idstr)
	if err != nil {
		return nil, err
	}

	dir, err := drv.GetDirectory(context.Background(), id, options...)
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

// storageOptionsFromGetQuery builds a new storage.GetOptions from gin query.
//
//nolint:cyclop,nolintlint // simple to follow.
func storageOptionsFromGetQuery(c *gin.Context) ([]storage.Option, error) {
	var options []storage.Option

	if value, ok := c.GetQuery("with_deleted"); ok {
		if withDeleted, err := strconv.ParseBool(value); err == nil {
			if withDeleted {
				options = append(options, storage.WithDeletedDirectories)
			}
		} else {
			return nil, err
		}
	}

	var (
		page  int
		limit int
	)

	if value, ok := c.GetQuery("page"); ok {
		if reqPage, err := strconv.Atoi(value); err == nil {
			if reqPage > 0 {
				page = reqPage
			}
		} else {
			return nil, err
		}
	}

	if value, ok := c.GetQuery("limit"); ok {
		if reqLimit, err := strconv.Atoi(value); err == nil {
			if reqLimit > 0 {
				limit = reqLimit
			}
		} else {
			return nil, err
		}
	}

	if page > 0 || limit > 0 {
		options = append(options, storage.Pagination(page, limit))
	}

	return options, nil
}

// storageOptionsToURLValues builds a new storage.GetOptions from gin query.
//
//nolint:cyclop,nolintlint // simple to follow.
func storageOptionsToURLValues(opts *storage.Options, page int) url.Values {
	values := make(url.Values)

	if opts.WithDeletedDirectories {
		values.Set("with_deleted", "true")
	}

	// Only when page is provided, include pagination details.
	if page > 0 {
		if reqPage := opts.GetPage(); reqPage >= 1 {
			values.Set("page", strconv.Itoa(page))
		}

		values.Set("limit", strconv.Itoa(opts.GetPageSize()))
	}

	return values
}

func paginationResponse(c *gin.Context, count int, options []storage.Option) v1.Pagination {
	opts := storage.BuildOptions(options)

	pagination := &v1.Pagination{
		Page:     opts.GetPage(),
		PageSize: opts.GetPageSize(),
	}

	// If the count is equal to the max size, then we'll assume there may be another page.
	// If the count is not equal, we'll assume we've reached the end and won't provide a next url.
	if count == opts.GetPageSize() {
		values := storageOptionsToURLValues(opts, opts.GetPage()+1)

		pagination.Links.Next = &v1.Link{HREF: buildURL(c, values).String()}
	}

	return *pagination
}

// buildURL returns a new *url.URL for the current page being requested, overwriting the values with the ones provided.
//
//nolint:cyclop // necessary complexity.
func buildURL(c *gin.Context, values url.Values) *url.URL {
	outURL := new(url.URL)

	if c != nil && c.Request != nil && c.Request.URL != nil {
		outURL.Path = c.Request.URL.Path
		outURL.RawQuery = values.Encode()

		// gin doesn't expose an easy way to check if the request came from a trusted proxy.
		// However the ClientIP will return the source ip instead of the remote ip if coming from a trusted proxy.
		// So we can compare the two, if they're the same, then we're either not behind a proxy or not behind a trusted proxy.
		if c.ClientIP() != c.RemoteIP() {
			if scheme := c.Request.Header.Get("X-Forwarded-Proto"); scheme != "" {
				outURL.Scheme = scheme
			}

			if host := c.Request.Header.Get("X-Forwarded-Host"); host != "" {
				outURL.Host = host
			}
		}

		if outURL.Scheme == "" {
			// Request.URL.Scheme is usually empty, however if it isn't we'll use that scheme.
			// If empty, we'll check if TLS was used, and if so, set the scheme as https.
			switch {
			case c.Request.URL.Scheme != "":
				outURL.Scheme = c.Request.URL.Scheme
			case c.Request.TLS != nil:
				outURL.Scheme = "https"
			default:
				outURL.Scheme = "http"
			}
		}

		if outURL.Host == "" {
			if host := c.Request.Host; host != "" {
				outURL.Host = host
			}
		}
	}

	return outURL
}
