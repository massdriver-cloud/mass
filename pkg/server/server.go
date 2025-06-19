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
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/cli/browser"
	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/container"
	"github.com/massdriver-cloud/mass/pkg/proxy"
	sb "github.com/massdriver-cloud/mass/pkg/server/bundle"
	sv "github.com/massdriver-cloud/mass/pkg/server/version"
	"github.com/massdriver-cloud/mass/pkg/templatecache"
	"github.com/massdriver-cloud/mass/pkg/version"
	"github.com/moby/moby/client"
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

func (b *BundleServer) Start(port string, launchBrowser bool) error {
	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	serverURL := "http://" + ln.Addr().String()

	slog.Info(fmt.Sprintf("Visit %s in your browser", serverURL))

	if launchBrowser {
		go openUIinBrowser(serverURL)
	}

	return b.httpServer.Serve(ln)
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

	// Routes to handle the UI
	http.Handle("/", originHeaderMiddleware(http.FileServer(http.Dir(filepath.Join(bundleUIDir, "public")))))
	http.Handle("/dist/", originHeaderMiddleware(http.FileServer(http.Dir(bundleUIDir))))
	http.Handle("/public/", originHeaderMiddleware(http.FileServer(http.Dir(bundleUIDir))))

	// The /deploy route wants the bundle.js file so redirect it to the index.html so it can download it again
	http.HandleFunc("/deploy", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		http.ServeFile(w, r, path.Join(bundleUIDir, "public/index.html"))
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_, err = w.Write([]byte("ok"))
		if err != nil {
			slog.Error("Error attempting to write healthcheck", "error", err.Error())
		}
	})

	// Route to handle the bundle files from the user's sytem - Any file in a bundle can be accessed
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
	http.Handle("/bundle/envs", originHeaderMiddleware(http.HandlerFunc(bundleHandler.GetEnvironmentVariables)))
	http.Handle("/bundle/params", originHeaderMiddleware(http.HandlerFunc(bundleHandler.Params)))

	// configHandler, err := config.NewHandler() //nolint:contextcheck
	// if err != nil {
	// 	slog.Error(err.Error())
	// 	os.Exit(1)
	// }

	// http.Handle("/config", originHeaderMiddleware(configHandler))

	http.Handle("/containers/logs", originHeaderMiddleware(http.HandlerFunc(containerHandler.StreamLogs)))
	http.Handle("/containers/list", originHeaderMiddleware(http.HandlerFunc(containerHandler.List)))
	http.Handle("/version", originHeaderMiddleware(http.HandlerFunc(sv.Latest)))
}

func originHeaderMiddleware(next http.Handler) http.Handler {
	accessControlHeaderName := "Access-Control-Allow-Origin"
	accessControlHeaderValue := "*"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			headers := w.Header()

			headers["Access-Control-Allow-Headers"] = r.Header["Access-Control-Request-Headers"]
			headers["Access-Control-Allow-Methods"] = []string{"GET, POST"}
			headers[accessControlHeaderName] = []string{accessControlHeaderValue}

			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set(accessControlHeaderName, accessControlHeaderValue)

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
	massDir, err := templatecache.GetOrCreateMassDir()
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

	return bundleUIDir, os.MkdirAll(bundleUIDir, os.ModePerm)
}

// sanitizeArchivePath from "G305: Zip Slip vulnerability" - stop naughty path traversal like ../..
func sanitizeArchivePath(d, t string) (v string, err error) {
	v = filepath.Join(d, t)
	if strings.HasPrefix(v, filepath.Clean(d)) {
		return v, nil
	}

	return "", fmt.Errorf("%s: %s", "content filepath is tainted", t)
}

func openUIinBrowser(serverURL string) {
	var iter int
	for {
		if iter > 3 {
			slog.Warn("Server never responded healthy")
			return
		}

		healthURL, err := url.JoinPath(serverURL, "health")
		if err != nil {
			slog.Error("Error creating healthcheck", "error", err.Error())
			return
		}

		// If the server isn't up it won't respond so use a timeout to try the request again
		ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancelFunc()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			slog.Error("Error creating healthcheck", "error", err.Error())
			return
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			if strings.HasSuffix(err.Error(), "context deadline exceeded") {
				// Log under Debug, if the server isn't up yet then the context timeout
				// would trigger an error here which isn't really useful
				slog.Debug("Error getting healthcheck", "error", err.Error())
			} else {
				slog.Warn("Error getting healthcheck", "error", err.Error())
			}
		} else {
			defer res.Body.Close()

			if res.StatusCode == http.StatusOK {
				if err = browser.OpenURL(serverURL); err != nil {
					slog.Error("Error trying to open browser", "error", err.Error())
				}
				return
			}
		}

		iter++
		time.Sleep(200 * time.Millisecond)
	}
}
