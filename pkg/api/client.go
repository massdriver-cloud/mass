package api

import (
	"net/http"

	"github.com/Khan/genqlient/graphql"
)

const Endpoint string = "https://api.massdriver.cloud/api/"

func NewClient(endpoint string, apiKey string) graphql.Client {
	c := http.Client{Transport: &authedTransport{wrapped: http.DefaultTransport, apiKey: apiKey}}
	return graphql.NewClient(endpoint, &c)
}

type authedTransport struct {
	wrapped http.RoundTripper
	apiKey  string
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("authorization", "Bearer "+t.apiKey)
	return t.wrapped.RoundTrip(req)
}
