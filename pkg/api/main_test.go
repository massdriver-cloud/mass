package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/Khan/genqlient/graphql"
)

const mockEndpoint string = "/graphql"

func mockClient(mux *http.ServeMux) graphql.Client {
	return graphql.NewClient(mockEndpoint, &http.Client{Transport: localRoundTripper{handler: mux}})
}

// localRoundTripper is an http.RoundTripper that executes HTTP transactions
// by using handler directly, instead of going over an HTTP connection.
type localRoundTripper struct {
	handler http.Handler
}

func (l localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	l.handler.ServeHTTP(w, req)
	return w.Result(), nil
}

func mustWrite(w io.Writer, s string) {
	_, err := io.WriteString(w, s)
	if err != nil {
		panic(err)
	}
}

func muxWithJSONResponse(response map[string]interface{}) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(mockEndpoint, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data, _ := json.Marshal(response)
		mustWrite(w, string(data))
	})

	return mux
}
