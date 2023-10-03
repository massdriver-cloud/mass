package cmd

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	_ "github.com/massdriver-cloud/mass/docs" // Init swagger docs
	"github.com/massdriver-cloud/mass/pkg/server"
	"github.com/spf13/cobra"
)

var programLevel = new(slog.LevelVar) // Info by default

func NewCmdServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the bundle development server",
		Long:  "Start the bundle development server. If no port is supplied an ephemeral port will be used",
		Run: func(cmd *cobra.Command, args []string) {
			runServer(cmd)
		},
		Args: cobra.NoArgs,
	}

	cmd.Flags().StringP("port", "p", "8080", "port for the server to listen on")
	cmd.Flags().StringP("directory", "d", "", "directory for the massdriver bundle, will default to the directory the server is ran from")
	cmd.Flags().String("log-level", "info", "Set the log level for the server. Options are [debug, info, warn, error]")

	return cmd
}

// @title						Massdriver API
// @description				Massdriver Bundle Development Server API
// @contact.url				https://github.com/massdriver-cloud/mass
// @contact.name				Massdriver
// @license.name				Apache 2.0
// @license.url				https://github.com/massdriver-cloud/mass/blob/main/LICENSE
// @host						127.0.0.1:8080
// @externalDocs.description	OpenAPI
// @externalDocs.url			https://swagger.io/resources/open-api/
func runServer(cmd *cobra.Command) {
	logLevel, err := cmd.Flags().GetString("log-level")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	setupLogging(logLevel)

	dir, err := cmd.Flags().GetString("directory")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	// Check we have a massdriver.yaml file available, if not error out.
	_, err = os.Stat(path.Join(dir, "massdriver.yaml"))
	if err != nil {
		slog.Error("massdriver.yaml file is not available in the specified directory")
		os.Exit(1)
	}

	port, err := cmd.Flags().GetString("port")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	// If no port is supplied grab an ephemeral port
	if port == "" {
		port = "0"
	}

	server, err := server.New(dir)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	server.RegisterHandlers()

	handleSignals(server)

	if err = server.Start(port); err != nil {
		// The signal handler will shutdown the server under a ctrl-c
		// so getting a ErrServerClosed here is expected
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error(err.Error())
		}
		slog.Info("Server is stopped")
	}
}

func setupLogging(level string) {
	switch strings.ToLower(level) {
	case "debug":
		programLevel.Set(slog.LevelDebug)
	case "info":
		programLevel.Set(slog.LevelInfo)
	case "warn":
		programLevel.Set(slog.LevelWarn)
	case "error":
		programLevel.Set(slog.LevelError)
	default:
		slog.Info("Unknown log level, setting to INFO", "level", level)
	}

	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})
	slog.SetDefault(slog.New(h))
}

func handleSignals(s *server.BundleServer) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(s *server.BundleServer) {
		for sig := range c {
			slog.Info("Shutting down", "signal", sig)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

			// If there are no errors here, the main func will race to exit potentially
			// before hitting the context cancel which is fine since we are already on the way out.
			if err := s.Stop(ctx); err != nil {
				if !errors.Is(err, http.ErrServerClosed) {
					slog.Error(err.Error())
				}
			}
			cancel()
			os.Exit(0)
		}
	}(s)
}
