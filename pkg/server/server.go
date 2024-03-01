package server

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/config"
	"github.com/massdriver-cloud/mass/pkg/container"
	"github.com/massdriver-cloud/mass/pkg/proxy"
	sb "github.com/massdriver-cloud/mass/pkg/server/bundle"
	sv "github.com/massdriver-cloud/mass/pkg/server/version"
	"github.com/massdriver-cloud/mass/pkg/templatecache"
	"github.com/massdriver-cloud/mass/pkg/version"
	"github.com/moby/moby/client"
	"github.com/spf13/afero"
	httpSwagger "github.com/swaggo/http-swagger"
)

type BundleServer struct {
	BaseDir   string
	Bundle    *bundle.Bundle
	DockerCli *client.Client

	httpServer *http.Server
}

func New(dir string) (*BundleServer, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.41"))
	if err != nil {
		return nil, fmt.Errorf("error creating docker client %w", err)
	}

	bundler, err := bundle.Unmarshal(dir)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling bundle %w", err)
	}

	server := &http.Server{ReadHeaderTimeout: 60 * time.Second}

	return &BundleServer{
		BaseDir:    dir,
		Bundle:     bundler,
		DockerCli:  cli,
		httpServer: server,
	}, nil
}

func (b *BundleServer) Start(port string) error {
	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	slog.Info(fmt.Sprintf("Visit http://%s in your browser", ln.Addr().String()))
	err = b.httpServer.Serve(ln)
	if err != nil {
		return err
	}
	return nil
}

func (b *BundleServer) Stop(ctx context.Context) error {
	return b.httpServer.Shutdown(ctx)
}

// RegisterHandlers registers with the DefaultServeMux to handle requests
func (b *BundleServer) RegisterHandlers(ctx context.Context) {
	bundleUIDir, err := setupUIDir()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	if err = getUIFiles(ctx, bundleUIDir); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		http.ServeFile(w, r, path.Join(bundleUIDir, "public/index.html"))
	})

	http.Handle("/dist/", originHeaderMiddleware(http.FileServer(http.Dir(bundleUIDir))))
	http.Handle("/public/", originHeaderMiddleware(http.FileServer(http.Dir(bundleUIDir))))

	http.Handle("/bundle-server/", http.StripPrefix("/bundle-server/", (originHeaderMiddleware(http.FileServer(http.Dir(b.BaseDir))))))

	http.HandleFunc("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://127.0.0.1:8080/swagger/doc.json"), // The url pointing to API definition
	))

	proxy, err := proxy.New("https://api.massdriver.cloud")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	http.Handle("/proxy/", originHeaderMiddleware(proxy))

	bundleHandler, err := sb.NewHandler(b.BaseDir)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	containerHandler := container.NewHandler(b.BaseDir, b.DockerCli)

	http.Handle("/bundle/build", originHeaderMiddleware(http.HandlerFunc(bundleHandler.Build)))
	http.Handle("/bundle/secrets", originHeaderMiddleware(http.HandlerFunc(bundleHandler.GetSecrets)))
	http.Handle("/bundle/connections", originHeaderMiddleware(http.HandlerFunc(bundleHandler.Connections)))
	http.Handle("/bundle/deploy", originHeaderMiddleware(http.HandlerFunc(containerHandler.Deploy)))

	configHandler, err := config.NewHandler() //nolint:contextcheck
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	http.Handle("/config", originHeaderMiddleware(configHandler))

	http.Handle("/containers/logs", originHeaderMiddleware(http.HandlerFunc(containerHandler.StreamLogs)))
	http.Handle("/containers/list", originHeaderMiddleware(http.HandlerFunc(containerHandler.List)))
	http.Handle("/version", originHeaderMiddleware(http.HandlerFunc(sv.Latest)))
}

func originHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			headers := w.Header()

			headers["Access-Control-Allow-Headers"] = r.Header["Access-Control-Request-Headers"]
			headers["Access-Control-Allow-Methods"] = []string{"GET, POST"}
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")

		next.ServeHTTP(w, r)
	})
}

func getUIFiles(ctx context.Context, baseDir string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, version.BundleBuilderUI, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	r, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(r)

	for {
		header, err1 := tarReader.Next()
		if err1 == io.EOF {
			break
		} else if err1 != nil {
			return err1
		}

		path, errS := sanitizeArchivePath(baseDir, header.Name)
		if errS != nil {
			return errS
		}

		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		if err = os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
			return err
		}

		file, createErr := os.Create(path)
		if createErr != nil {
			return createErr
		}

		defer file.Close()

		// Ignore the gosec linting, we are pulling from our repo only
		_, err = io.Copy(file, tarReader) // #nosec G110
		if err != nil {
			return err
		}
	}
	return nil
}

// setupUIDir creates the base dir for the bundle-ui based off the mass dir
func setupUIDir() (string, error) {
	localFs := afero.NewOsFs()
	massDir, err := templatecache.GetOrCreateMassDir(localFs)
	if err != nil {
		return "", err
	}

	bundleUIDir := path.Join(massDir, "bundle-ui")

	// TODO: Add some smarts so we know what version we are on and don't always wipe the dir
	if _, err = os.Stat(bundleUIDir); err == nil {
		slog.Debug("Cleaning up UI dir")
		if err = os.RemoveAll(bundleUIDir); err != nil {
			slog.Warn("Error cleaning up UI dir", "error", err)
		}
	}

	return bundleUIDir, localFs.MkdirAll(bundleUIDir, os.ModePerm)
}

// sanitizeArchivePath from "G305: Zip Slip vulnerability" - stop naughty path traversal like ../..
func sanitizeArchivePath(d, t string) (v string, err error) {
	v = filepath.Join(d, t)
	if strings.HasPrefix(v, filepath.Clean(d)) {
		return v, nil
	}

	return "", fmt.Errorf("%s: %s", "content filepath is tainted", t)
}
