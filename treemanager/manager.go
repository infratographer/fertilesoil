package treemanager

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"entgo.io/ent/dialect/sql/schema"

	// The runtime import is needed to register the hooks
	"github.com/JAORMX/fertilesoil/ent/directory"
	_ "github.com/JAORMX/fertilesoil/ent/runtime"
	"github.com/gin-gonic/gin"

	"github.com/JAORMX/fertilesoil/ent"
)

type ServerConfig struct {
	SQLDriver        string
	ConnectionString string
	BootStrap        bool
}

func (cfg *ServerConfig) Run(ctx context.Context) error {
	cli, err := ent.Open(cfg.SQLDriver, cfg.ConnectionString)
	if err != nil {
		return fmt.Errorf("failed connecting to database: %w", err)
	}

	defer cli.Close()

	if cfg.BootStrap {
		// Run migration.
		err = cli.Schema.Create(ctx, schema.WithAtlas(true))
		if err != nil {
			return fmt.Errorf("failed creating schema resources: %w", err)
		}

		// Create a new directory.
		var err error
		_, err = cli.Directory.Create().SetName("root").SetIsRoot(true).Save(ctx)
		if err != nil {
			return fmt.Errorf("failed creating root directory: %w", err)
		}
	}

	r := gin.Default()

	r.GET("/api/v1/directory", func(c *gin.Context) {
		dirs, err := cli.Directory.Query().All(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, dirs)
	})

	r.GET("/api/v1/directory/:id", func(c *gin.Context) {
		idstr := c.Param("id")
		id, converr := strconv.Atoi(idstr)
		if converr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": converr.Error()})
			return
		}
		dir, err := cli.Directory.Query().Where(directory.ID(id)).Only(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, dir)
	})

	r.POST("/api/v1/directory/:id", func(c *gin.Context) {
		idstr := c.Param("id")
		id, converr := strconv.Atoi(idstr)
		if converr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": converr.Error()})
			return
		}
		dir, err := cli.Directory.Query().Where(directory.ID(id)).Only(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var newDir *ent.Directory
		newDir, err = cli.Directory.Create().SetName("child-dir").SetParent(dir).Save(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, newDir)
	})

	r.GET("/api/v1/directory/:id/children", func(c *gin.Context) {
		idstr := c.Param("id")
		id, converr := strconv.Atoi(idstr)
		if converr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": converr.Error()})
			return
		}

		children, cerr := recurseChildren(cli, id)
		if cerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, children)
	})

	r.GET("/api/v1/directory/:id/ancestors", func(c *gin.Context) {
		idstr := c.Param("id")
		id, converr := strconv.Atoi(idstr)
		if converr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": converr.Error()})
			return
		}

		// The recurseAncestors call already does a query to get the directory
		// so we don't need to do it here.
		ancestors, err := recurseAncestors(cli, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, ancestors)
	})

	r.GET("/api/v1/directory/:id/ancestors/:untilid", func(c *gin.Context) {
		idstr := c.Param("id")
		id, converr := strconv.Atoi(idstr)
		if converr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": converr.Error()})
			return
		}

		untilidstr := c.Param("untilid")
		untilid, converr := strconv.Atoi(untilidstr)
		if converr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": converr.Error()})
			return
		}

		// determine if ancestor is even in the tree
		_, err = cli.Directory.Query().Where(directory.ID(untilid)).WithParent().Only(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ancestors, err := recurseAncestorsUntil(cli, id, untilid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, ancestors)
	})

	return r.Run()
}

func recurseChildren(cli *ent.Client, id int) ([]*ent.Directory, error) {
	children, err := cli.Directory.Query().Where(directory.HasParentWith(directory.ID(id))).All(context.Background())
	if err != nil {
		return nil, err
	}

	for _, child := range children {
		recursedchildren, err := recurseChildren(cli, child.ID)
		if err != nil {
			return nil, err
		}

		children = append(children, recursedchildren...)
	}

	return children, nil
}

func recurseAncestors(cli *ent.Client, id int) ([]*ent.Directory, error) {
	anc, _, err := recurseAncestry(cli, id, false, 0)
	return anc, err
}

func recurseAncestorsUntil(cli *ent.Client, id int, untilid int) ([]*ent.Directory, error) {
	anc, found, err := recurseAncestry(cli, id, true, untilid)
	if !found {
		return nil, fmt.Errorf("Ancestor %d not found in ancestry", untilid)
	}
	return anc, err
}

// recurseAncestry returns all ancestors of a directory until a specific directory is reached
// this is useful for determining if a directory is a child of another directory
func recurseAncestry(
	cli *ent.Client,
	id int,
	fullAncestry bool,
	untilid int,
) ([]*ent.Directory, bool, error) {
	dir, err := cli.Directory.Query().Where(directory.ID(id)).WithParent().Only(context.Background())
	if err != nil {
		return nil, false, err
	}

	if dir.Edges.Parent == nil {
		return []*ent.Directory{}, !fullAncestry, nil
	}

	ancestors := []*ent.Directory{dir.Edges.Parent}

	if dir.Edges.Parent.ID == untilid && fullAncestry {
		return ancestors, true, nil
	}
	anc, ancestorFound, err := recurseAncestry(cli, dir.Edges.Parent.ID, fullAncestry, untilid)
	if err != nil {
		return nil, false, err
	}

	if ancestorFound || !fullAncestry {
		return append(ancestors, anc...), true, nil
	}
	return []*ent.Directory{}, false, nil
}
