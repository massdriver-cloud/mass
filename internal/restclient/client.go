package restclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

type PublishPost struct {
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Type              string                 `json:"type"`
	SourceURL         string                 `json:"source_url"`
	Access            string                 `json:"access"`
	ArtifactsSchema   map[string]interface{} `json:"artifacts_schema"`
	ConnectionsSchema map[string]interface{} `json:"connections_schema"`
	ParamsSchema      map[string]interface{} `json:"params_schema"`
	UISchema          map[string]interface{} `json:"ui_schema"`
	OperatorGuide     []byte                 `json:"operator_guide,omitempty"`
	AppSpec           map[string]interface{} `json:"app,omitempty"`
	Runbook           []byte                 `json:"runbook,omitempty"`
}

type PublishResponse struct {
	UploadLocation string `json:"upload_location"`
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

func (c *MassdriverClient) PublishBundle(request PublishPost) (string, error) {
	bodyBytes, err := json.Marshal(request)

	if err != nil {
		return "", err
	}

	ctx := context.Background()
	req := NewRequest("PUT", "bundles", bytes.NewBuffer(bodyBytes))

	resp, err := c.Do(&ctx, req)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.Status != "200 OK" {
		fmt.Println(string(respBodyBytes))
		return "", errors.New("received non-200 response from Massdriver: " + resp.Status)
	}

	var respBody PublishResponse
	err = json.Unmarshal(respBodyBytes, &respBody)
	if err != nil {
		return "", err
	}

	return respBody.UploadLocation, nil
}
