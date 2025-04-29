package definition

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path"

	"github.com/massdriver-cloud/mass/pkg/restclient"
)

func Get(c *restclient.MassdriverClient, definitionType string) (map[string]interface{}, error) {
	var definition map[string]interface{}

	endpoint := path.Join("artifact-definitions", definitionType)

	req := restclient.NewRequest("GET", endpoint, nil)

	ctx := context.Background()
	resp, err := c.Do(&ctx, req)

	if err != nil {
		return definition, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return definition, errors.New("received non-200 response from Massdriver: " + resp.Status + " " + definitionType)
	}

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return definition, err
	}

	err = json.Unmarshal(respBodyBytes, &definition)
	if err != nil {
		return definition, err
	}

	return definition, nil
}
