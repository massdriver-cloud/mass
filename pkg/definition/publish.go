package definition

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/massdriver-cloud/mass/pkg/restclient"
)

func (art *Definition) Publish(c *restclient.MassdriverClient) error {
	bodyBytes, err := json.Marshal(*art)
	if err != nil {
		return err
	}

	req := restclient.NewRequest("PUT", "artifact-definitions", bytes.NewBuffer(bodyBytes))
	ctx := context.Background()
	resp, err := c.Do(&ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.Status != "200 OK" {
		respBodyBytes, err2 := io.ReadAll(resp.Body)
		if err2 != nil {
			return err2
		}
		fmt.Println(string(respBodyBytes))
		return errors.New("received non-200 response from Massdriver: " + resp.Status)
	}

	return nil
}
