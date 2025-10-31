package api

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

type Server struct {
	Version string `json:"version"`
	Mode    string `json:"mode"`
	AppURL  string `json:"appUrl"`
}

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
