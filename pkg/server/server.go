package server

import (
	"embed"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/massdriver-cloud/mass/pkg/proxy"
)

var (
	//go:embed site
	res   embed.FS
	pages = map[string]string{
		"/hello-agent": "site/index.html",
	}
)

// RegisterServerHandler registers with the DefaultServeMux to handle requests
func RegisterServerHandler(dir string) {
	// Register a FileServer that will give access to the assets dir
	// http.Handle("/site/assets/", http.FileServer(http.FS(res)))

	http.Handle("/", http.FileServer(http.Dir(dir)))

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

	http.Handle("/proxy/", proxy)
}
