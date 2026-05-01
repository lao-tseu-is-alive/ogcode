package cli

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/ogcode/ogcode/internal/server"
	"github.com/spf13/cobra"
)

var port int

var rootCmd = &cobra.Command{
	Use:   "ogcode",
	Short: "Agentic coding assistant with web UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		return serve(cmd, args)
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the ogcode server",
	RunE:  serve,
}

func init() {
	rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}

func serve(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	srv := server.New(port, dir)
	return srv.Start()
}

func Execute() error {
	_ = godotenv.Load()
	return rootCmd.Execute()
}