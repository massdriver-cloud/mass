package server

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/config"
	"github.com/massdriver-cloud/mass/pkg/container"
	"github.com/massdriver-cloud/mass/pkg/proxy"
	"github.com/moby/moby/client"
	"github.com/spf13/afero"
	httpSwagger "github.com/swaggo/http-swagger"
)

var (
	//go:embed site
	res   embed.FS
	pages = map[string]string{
		"/hello-agent": "site/index.html",
	}
)

type BundleServer struct {
	BaseDir   string
	Bundle    *bundle.Bundle
	DockerCli *client.Client
	Fs        afero.Fs

	httpServer *http.Server
}

func New(dir string) (*BundleServer, error) {
	fs := afero.NewOsFs()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.41"))
	if err != nil {
		return nil, fmt.Errorf("error creating docker client %w", err)
	}

	bundler, err := bundle.UnmarshalBundle(dir, fs)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling bundle %w", err)
	}

	server := &http.Server{ReadHeaderTimeout: 60 * time.Second}

	return &BundleServer{
		BaseDir:    dir,
		Bundle:     bundler,
		DockerCli:  cli,
		Fs:         fs,
		httpServer: server,
	}, nil
}

func (b *BundleServer) Start(port string) error {
	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	slog.Info(fmt.Sprintf("Visit http://%s/hello-agent in your browser", ln.Addr().String()))
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
func (b *BundleServer) RegisterHandlers() {
	// Register a FileServer that will give access to the assets dir
	// http.Handle("/site/assets/", http.FileServer(http.FS(res)))

	http.Handle("/", originHeaderMiddleware(http.FileServer(http.Dir(b.BaseDir))))

	http.HandleFunc("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://127.0.0.1:8080/swagger/doc.json"), // The url pointing to API definition
	))

	// Register the handler func to serve the html page
	http.HandleFunc("/hello-agent", func(w http.ResponseWriter, r *http.Request) {
		page, ok := pages[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		tpl, err := template.ParseFS(res, page)
		if err != nil {
			log.Printf("page %s not found in pages cache...", r.RequestURI)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		data := map[string]interface{}{
			"userAgent": r.UserAgent(),
		}
		if err = tpl.Execute(w, data); err != nil {
			return
		}
	})

	proxy, err := proxy.New("https://api.massdriver.cloud")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	http.Handle("/proxy/", originHeaderMiddleware(proxy))

	bundleHandler, err := bundle.NewHandler(b.BaseDir)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	http.Handle("/bundle/secrets", originHeaderMiddleware(http.HandlerFunc(bundleHandler.GetSecrets)))
	http.Handle("/bundle/connections", originHeaderMiddleware(http.HandlerFunc(bundleHandler.Connections)))

	configHandler, err := config.NewHandler()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	http.Handle("/config", originHeaderMiddleware(configHandler))

	http.Handle("/containers/logs", originHeaderMiddleware(http.HandlerFunc(container.StreamLogs)))
	http.Handle("/containers/list", originHeaderMiddleware(http.HandlerFunc(container.List)))
}

func originHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}
