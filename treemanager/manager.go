package treemanager

import (
	"context"
	"fmt"
	"log"
	"net"

	"entgo.io/ent/dialect/sql/schema"
	"google.golang.org/grpc"

	"github.com/JAORMX/fertilesoil/ent"
	"github.com/JAORMX/fertilesoil/ent/proto/entpb"
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
		err = cli.Schema.Create(ctx, schema.WithAtlas(true), schema.WithGlobalUniqueID(true))
		if err != nil {
			return fmt.Errorf("failed creating schema resources: %w", err)
		}
	}

	svc := entpb.NewDirectoryService(cli)

	server := grpc.NewServer()

	entpb.RegisterDirectoryServiceServer(server, svc)

	// Open port 5000 for listening to traffic.
	lis, err := net.Listen("tcp", ":5000")
	if err != nil {
		log.Fatalf("failed listening: %s", err)
	}

	// Listen for traffic indefinitely.
	if err := server.Serve(lis); err != nil {
		log.Fatalf("server ended: %s", err)
	}

	return nil
}
