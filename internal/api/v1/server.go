package api

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

// Server holds metadata about the Massdriver server instance.
type Server struct {
	AppURL  string `json:"appUrl" mapstructure:"appUrl"`
	Version string `json:"version" mapstructure:"version"`
	Mode    string `json:"mode" mapstructure:"mode"`
}

// GetServer retrieves server metadata from the Massdriver API.
func GetServer(ctx context.Context, mdClient *client.Client) (*Server, error) {
	response, err := getServer(ctx, mdClient.GQL)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	return toServer(response.Server)
}

func toServer(s any) (*Server, error) {
	server := Server{}
	if err := mapstructure.Decode(s, &server); err != nil {
		return nil, fmt.Errorf("failed to decode server: %w", err)
	}
	return &server, nil
}
