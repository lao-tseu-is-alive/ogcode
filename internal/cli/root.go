package cli

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/prasenjeet-symon/ogcode/internal/server"
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

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Start ogcode in Plan Mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		return serveWithMode(cmd, args, server.ModePlan)
	},
}

func init() {
	rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	planCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(planCmd)
}

func setupLogging() {
	level := slog.LevelInfo
	levelStr := strings.ToLower(strings.TrimSpace(os.Getenv("OGCODE_LOG_LEVEL")))
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	format := strings.ToLower(strings.TrimSpace(os.Getenv("OGCODE_LOG_FORMAT")))
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level, AddSource: level <= slog.LevelDebug}

	switch format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))

	slog.Info("logging initialized", "level", level, "format", format)
}

func serve(cmd *cobra.Command, args []string) error {
	return serveWithMode(cmd, args, server.ModeBuild)
}

func serveWithMode(cmd *cobra.Command, args []string, mode server.ServerMode) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	srv := server.New(port, dir, mode)
	return srv.Start()
}

func Execute() error {
	_ = godotenv.Load()
	setupLogging()
	return rootCmd.Execute()
}
