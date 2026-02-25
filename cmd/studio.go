package cmd

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/massdriver-cloud/mass/docs" // Init swagger docs
	"github.com/massdriver-cloud/mass/pkg/studio"
	"github.com/spf13/cobra"
)

var studioProgramLevel = new(slog.LevelVar) // Info by default

func NewCmdStudio() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "studio",
		Short: "Start the local development studio",
		Long: `Start the local development studio for bundles and artifact definitions.

The studio recursively scans the specified directory for massdriver.yaml files and
provides a web UI for:
- Previewing and configuring bundles
- Viewing and editing artifact definitions
- Building bundles
- Managing connections and parameters

Example:
  mass studio                    # Start studio in current directory
  mass studio -d ./my-bundles    # Start studio in specific directory
  mass studio --browser          # Auto-launch browser`,
		Run: func(cmd *cobra.Command, args []string) {
			runStudio(cmd)
		},
		Args: cobra.NoArgs,
	}

	cmd.Flags().StringP("port", "p", "8080", "port for the studio to listen on")
	cmd.Flags().StringP("directory", "d", ".", "directory to scan for massdriver.yaml files")
	cmd.Flags().String("log-level", "info", "set the log level [debug, info, warn, error]")
	cmd.Flags().Bool("browser", false, "launch browser window after starting")
	cmd.Flags().String("ui-dir", "", "serve UI from local directory instead of downloading (for development)")

	return cmd
}

// @title						Massdriver Studio API
// @description				Massdriver Local Development Studio API
// @contact.url				https://github.com/massdriver-cloud/mass
// @contact.name				Massdriver
// @license.name				Apache 2.0
// @license.url				https://github.com/massdriver-cloud/mass/blob/main/LICENSE
// @host						127.0.0.1:8080
// @externalDocs.description	OpenAPI
// @externalDocs.url			https://swagger.io/resources/open-api/
func runStudio(cmd *cobra.Command) {
	logLevel, err := cmd.Flags().GetString("log-level")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	setupStudioLogging(logLevel)

	dir, err := cmd.Flags().GetString("directory")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	// Default to current directory if not specified
	if dir == "" {
		dir = "."
	}

	port, err := cmd.Flags().GetString("port")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	// If no port is supplied, use ephemeral port
	if port == "" {
		port = "0"
	}

	uiDir, err := cmd.Flags().GetString("ui-dir")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	s, err := studio.New(dir)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	ctx := context.Background()

	s.RegisterHandlers(ctx, uiDir)

	handleStudioSignals(ctx, s)

	launchBrowser, err := cmd.Flags().GetBool("browser")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	if err = s.Start(port, launchBrowser); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error(err.Error())
		}
		slog.Info("Studio is stopped")
	}
}

func setupStudioLogging(level string) {
	switch strings.ToLower(level) {
	case "debug":
		studioProgramLevel.Set(slog.LevelDebug)
	case "info":
		studioProgramLevel.Set(slog.LevelInfo)
	case "warn":
		studioProgramLevel.Set(slog.LevelWarn)
	case "error":
		studioProgramLevel.Set(slog.LevelError)
	default:
		slog.Info("Unknown log level, setting to INFO", "level", level)
	}

	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: studioProgramLevel})
	slog.SetDefault(slog.New(h))
}

func handleStudioSignals(ctx context.Context, s *studio.Studio) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(s *studio.Studio) {
		for sig := range c {
			slog.Info("Shutting down", "signal", sig)
			ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)

			if err := s.Stop(ctxTimeout); err != nil {
				if !errors.Is(err, http.ErrServerClosed) {
					slog.Error(err.Error())
				}
			}
			cancel()
			os.Exit(0)
		}
	}(s)
}
