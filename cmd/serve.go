/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/spf13/cobra"

	"github.com/JAORMX/fertilesoil/treemanager"
)

// serveCmd represents the treemanager command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ts, err := testserver.NewTestServer()
		if err != nil {
			return fmt.Errorf("failed to start test server: %w", err)
		}

		defer ts.Stop()

		if err := ts.WaitForInit(); err != nil {
			return fmt.Errorf("failed to wait for test server to initialize: %w", err)
		}

		// catch SIGTERM and SIGINT
		ctx, cancel := context.WithCancel(cmd.Context())
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		defer func() {
			signal.Stop(c)
			cancel()
		}()
		go func() {
			select {
			case <-c:
				cancel()
			case <-ctx.Done():
			}
		}()

		sc := treemanager.ServerConfig{
			SQLDriver:        "postgres",
			ConnectionString: ts.PGURL().String(),
			BootStrap:        true,
		}

		return sc.Run(ctx)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// treemanagerCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// treemanagerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
