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

func Publish(c *restclient.MassdriverClient, in io.Reader) error {

	byteValue, _ := io.ReadAll(in)

	// attempt to unmarshall to make sure it's valid JSON
	// TODO: use JSON schema to validate
	var artdef Definition
	if jsonErr := json.Unmarshal(byteValue, &artdef); jsonErr != nil {
		return jsonErr
	}

	req := restclient.NewRequest("PUT", "artifact-definitions", bytes.NewBuffer(byteValue))
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
