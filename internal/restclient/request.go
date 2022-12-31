package restclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Request struct {
	Method string
	Path   string
	Body   io.Reader
}

func NewRequest(method string, path string, body io.Reader) *Request {
	req := new(Request)

	req.Method = method
	req.Path = path
	req.Body = body

	return req
}

func (req *Request) ToHTTPRequest(ctx context.Context, c *MassdriverClient) (*http.Request, error) {
	url, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}

	url.Path = req.Path

	// TODO: is there a better place to set this context?
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url.String(), req.Body)
	if err != nil {
		return nil, err
	}

	if c.apiKey != "" {
		httpReq.Header.Set("X-Md-Api-Key", c.apiKey)
	} else {
		fmt.Println("Warning: API Key not specified")
	}
	// for now assuming everything is json
	httpReq.Header.Set("Content-Type", "application/json")

	return httpReq, nil
}
