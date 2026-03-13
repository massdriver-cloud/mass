// Package proxy provides an HTTP reverse proxy for Massdriver API requests.
package proxy

import (
	"log/slog"
	"net/http/httputil"
	"net/url"
	"strings"
)

// New creates a reverse proxy that forwards requests to the given URL.
func New(proxyURL string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)

			r.Out.URL.Path = strings.TrimPrefix(r.Out.URL.Path, "/proxy")
			slog.Debug("Proxying request", "path", r.Out.URL.Path, "method", r.Out.Method)
		},
	}

	return proxy, nil
}
