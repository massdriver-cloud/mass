package restclient

import (
	"context"
	"net/http"
	"os"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type MassdriverClient struct {
	Client  HTTPClient
	baseURL string
	apiKey  string
}

const MassdriverBaseURL = "https://api.massdriver.cloud"

func NewClient() *MassdriverClient {
	c := new(MassdriverClient)

	c.Client = http.DefaultClient
	c.baseURL = getBaseURL()
	c.apiKey = getAPIKey()

	return c
}

// eventually this could walk through multiple sources (environment, then config file, etc)
func getAPIKey() string {
	return os.Getenv("MASSDRIVER_API_KEY")
}

func getBaseURL() string {
	if endpoint, ok := os.LookupEnv("MASSDRIVER_URL"); ok {
		return endpoint
	}
	return MassdriverBaseURL
}

func (c *MassdriverClient) WithAPIKey(apiKey string) *MassdriverClient {
	c.apiKey = apiKey
	return c
}

func (c *MassdriverClient) WithBaseURL(endpoint string) *MassdriverClient {
	c.baseURL = endpoint
	return c
}

func (c *MassdriverClient) Do(ctx *context.Context, req *Request) (*http.Response, error) {
	httpReq, err := req.ToHTTPRequest(*ctx, c)
	if err != nil {
		return nil, err
	}

	return c.Client.Do(httpReq)
}
